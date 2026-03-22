package processor

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultMaxWorkers = 4
	PollInterval      = 10 * time.Second
)

type JobStoreWithNotify interface {
	JobStore
	SubscribeOnJob() chan struct{}
	UnsubscribeOnJob(ch chan struct{})
}

type WorkerPool struct {
	maxWorkers    int
	processor     Processor
	jobStore      JobStoreWithNotify
	imageStore    ImageStore
	kafkaProducer KafkaNotifier
	logger        *slog.Logger
	wg            sync.WaitGroup
	sem           chan struct{}
	notifyCh      chan struct{}
	stopCh        chan struct{}
	pending       int64
}

func NewWorkerPool(
	processor Processor,
	jobStore JobStoreWithNotify,
	imageStore ImageStore,
	kafkaProducer KafkaNotifier,
	logger *slog.Logger,
	maxWorkers int,
) *WorkerPool {
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers
	}

	return &WorkerPool{
		maxWorkers:    maxWorkers,
		processor:     processor,
		jobStore:      jobStore,
		imageStore:    imageStore,
		kafkaProducer: kafkaProducer,
		logger:        logger,
		sem:           make(chan struct{}, maxWorkers),
		notifyCh:      jobStore.SubscribeOnJob(),
		stopCh:        make(chan struct{}),
	}
}

// Run starts the worker pool with notification-based processing and polling fallback
func (p *WorkerPool) Run() {
	p.logger.Info("Starting worker pool", "maxWorkers", p.maxWorkers)

	// Initialize pending counter with current job count
	count, err := p.jobStore.CountPendingJobs(context.Background())
	if err != nil {
		p.logger.Warn("Failed to count pending jobs", "error", err)
	} else {
		atomic.StoreInt64(&p.pending, int64(count))
		p.logger.Info("Pending jobs initialized", "count", count)
	}

	p.wg.Add(1)
	go p.pollingLoop()

	for {
		select {
		case <-p.notifyCh:
			p.logger.Debug("Job notification received")
			atomic.AddInt64(&p.pending, 1)
			p.tryStartWorker()
		case <-p.stopCh:
			p.logger.Info("Worker pool stopping")
			p.cleanup()
			return
		}
	}
}

// Stop stops the worker pool gracefully
func (p *WorkerPool) Stop() {
	close(p.stopCh)
	p.wg.Wait()
}

func (p *WorkerPool) tryStartWorker() bool {
	if atomic.LoadInt64(&p.pending) == 0 {
		return false
	}
	select {
	case p.sem <- struct{}{}:
		atomic.AddInt64(&p.pending, -1)
		p.wg.Add(1)
		p.logger.Debug("Starting worker")
		go p.runWorker()
		return true
	default:
		return false
	}
}

func (p *WorkerPool) runWorker() {
	defer func() {
		<-p.sem
		p.wg.Done()
		p.logger.Debug("Worker finished")

		if atomic.LoadInt64(&p.pending) > 0 {
			go func() {
				time.Sleep(time.Millisecond)
				p.tryStartWorker()
			}()
		}
	}()

	worker := NewWorker(p.processor, p.jobStore, p.imageStore, p.kafkaProducer, p.logger)
	if err := worker.Run(); err != nil {
		p.logger.Debug("Worker error", "error", err)
	}
}

// pollingLoop is a fallback mechanism that periodically checks for pending jobs
func (p *WorkerPool) pollingLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.tryStartWorker()
		case <-p.stopCh:
			return
		}
	}
}

func (p *WorkerPool) cleanup() {
	p.jobStore.UnsubscribeOnJob(p.notifyCh)
}
