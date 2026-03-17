package processor

import (
	"log/slog"
	"sync"
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
}

func NewWorkerPool(processor Processor, jobStore JobStoreWithNotify, imageStore ImageStore, kafkaProducer KafkaNotifier, logger *slog.Logger, maxWorkers int) *WorkerPool {
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

	p.wg.Add(1)
	go p.pollingLoop()

	for {
		select {
		case <-p.notifyCh:
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

func (p *WorkerPool) tryStartWorker() {
	select {
	case p.sem <- struct{}{}:
		p.wg.Add(1)
		go p.runWorker()
	default:
		// All workers busy
	}
}

func (p *WorkerPool) runWorker() {
	defer func() {
		<-p.sem
		p.wg.Done()
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
