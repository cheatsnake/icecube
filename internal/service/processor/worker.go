package processor

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cheatsnake/icm/internal/domain/image"
	"github.com/cheatsnake/icm/internal/domain/jobs"
	"github.com/cheatsnake/icm/internal/domain/processing"
)

type Processor interface {
	Process(imagePath string, options *processing.Options) (string, error)
}

type JobStore interface {
	AcquireJob(ctx context.Context) (*jobs.Job, error)
	ReleaseJobs(ctx context.Context, lease time.Duration) error
	UpdateJob(ctx context.Context, job *jobs.Job) error
	UpdateTasks(ctx context.Context, tasks []*jobs.Task) error
}

type ImageStore interface {
	UploadImage(ctx context.Context, r io.Reader) (*image.Variant, error)
	DownloadImage(ctx context.Context, id string) (io.ReadCloser, error)
	GetMetadataByID(ctx context.Context, id string) (*image.Variant, error)
}

type Worker struct {
	id         string
	processor  Processor
	jobStore   JobStore
	imageStore ImageStore
	logger     *slog.Logger
}

func NewWorker(id string, processor Processor, jobStore JobStore, imageStore ImageStore, logger *slog.Logger) *Worker {
	return &Worker{
		id:         id,
		processor:  processor,
		jobStore:   jobStore,
		imageStore: imageStore,
		logger:     logger,
	}
}

// Run acquires one job and processes it
func (w *Worker) Run() error {
	ctx := context.Background()

	job, err := w.jobStore.AcquireJob(ctx)
	if err != nil {
		return err
	}
	if job == nil {
		return nil
	}

	tmpDir, err := w.createTempDir(job)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	originalPath, err := w.downloadOriginal(ctx, job, tmpDir)
	if err != nil {
		return err
	}

	processedTasks, procErrs := w.processTasks(ctx, job, originalPath)

	// if any processing errors -> mark job as failed
	if len(procErrs) > 0 {
		// pick first error to return (consistent with previous behavior)
		first := procErrs[0]
		w.logger.Error("processing error", "message", first.Error())
		if err := w.markJobFailed(ctx, job); err != nil {
			// if failing to mark job failed, return that error (previously code returned update error)
			return err
		}
		return first
	}

	// sanity check: no completed tasks
	if len(processedTasks) == 0 {
		if err := w.markJobFailed(ctx, job); err != nil {
			return fmt.Errorf("no tasks were completed and failed to update job: %w", err)
		}
		return fmt.Errorf("no tasks were completed")
	}

	err = w.jobStore.UpdateTasks(ctx, processedTasks)
	if err != nil {
		return w.markJobFailed(ctx, job)
	}

	job.Status = jobs.JobStatusCompleted
	job.LockedAt = nil
	if err := w.jobStore.UpdateJob(ctx, job); err != nil {
		return err
	}

	return nil
}

func (w *Worker) createTempDir(job *jobs.Job) (string, error) {
	tmpDir, err := os.MkdirTemp("", "work-*")
	if err != nil {
		go w.releaseJob(context.Background(), job)
		w.logger.Warn("Failed to create temp directory for job " + job.ID)
		return "", err
	}
	return tmpDir, nil
}

func (w *Worker) downloadOriginal(ctx context.Context, job *jobs.Job, tmpDir string) (string, error) {
	original, err := w.imageStore.GetMetadataByID(ctx, job.OriginalID)
	if err != nil {
		go w.releaseJob(context.Background(), job)
		w.logger.Warn("Failed to get original image metadata for job " + job.ID)
		return "", err
	}

	originalPath := filepath.Join(tmpDir, "original."+string(original.Format))
	reader, err := w.imageStore.DownloadImage(ctx, job.OriginalID)
	if err != nil {
		go w.releaseJob(context.Background(), job)
		w.logger.Warn("Failed to download original image for job " + job.ID)
		return "", err
	}
	defer reader.Close()

	originalFile, err := os.Create(originalPath)
	if err != nil {
		go w.releaseJob(context.Background(), job)
		w.logger.Warn("Failed to create original file for job " + job.ID)
		return "", err
	}
	_, err = io.Copy(originalFile, reader)
	if errClose := originalFile.Close(); errClose != nil && err == nil {
		err = errClose
	}
	if err != nil {
		go w.releaseJob(context.Background(), job)
		w.logger.Warn("Failed to copy original image for job " + job.ID)
		return "", err
	}

	return originalPath, nil
}

func (w *Worker) processTasks(ctx context.Context, job *jobs.Job, originalPath string) ([]*jobs.Task, []error) {
	var wg sync.WaitGroup
	errorsCh := make(chan error, len(job.Tasks))
	resultsCh := make(chan *jobs.Task, len(job.Tasks))

	for _, t := range job.Tasks {
		wg.Add(1)
		go func(task *jobs.Task) {
			defer wg.Done()

			processedPath, err := w.processor.Process(originalPath, task.Options)
			if err != nil {
				errorsCh <- err
				return
			}
			defer func() {
				_ = os.Remove(processedPath)
			}()

			f, err := os.Open(processedPath)
			if err != nil {
				errorsCh <- err
				return
			}
			defer f.Close()

			variant, err := w.imageStore.UploadImage(ctx, f)
			if err != nil {
				errorsCh <- err
				return
			}

			task.Complete(variant.ID)
			resultsCh <- task
		}(t)
	}

	wg.Wait()
	close(errorsCh)
	close(resultsCh)

	var errs []error
	for e := range errorsCh {
		errs = append(errs, e)
	}

	var results []*jobs.Task
	for r := range resultsCh {
		results = append(results, r)
	}

	return results, errs
}

func (w *Worker) markJobFailed(ctx context.Context, job *jobs.Job) error {
	job.Status = jobs.JobStatusFailed
	job.LockedAt = nil
	if err := w.jobStore.UpdateJob(ctx, job); err != nil {
		return err
	}
	return nil
}

func (w *Worker) releaseJob(ctx context.Context, job *jobs.Job) error {
	job.Status = jobs.JobStatusPending
	job.LockedAt = nil
	if err := w.jobStore.UpdateJob(ctx, job); err != nil {
		return err
	}
	return nil
}
