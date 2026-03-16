package processor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cheatsnake/icecube/internal/domain/errs"
	"github.com/cheatsnake/icecube/internal/domain/image"
	"github.com/cheatsnake/icecube/internal/domain/jobs"
	"github.com/cheatsnake/icecube/internal/domain/processing"
	"github.com/cheatsnake/icecube/internal/pkg/fs"
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
	UploadImage(ctx context.Context, r io.Reader, name string, size int64) (*image.Variant, error)
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

func NewWorker(processor Processor, jobStore JobStore, imageStore ImageStore, logger *slog.Logger) *Worker {
	return &Worker{
		processor:  processor,
		jobStore:   jobStore,
		imageStore: imageStore,
		logger:     logger,
	}
}

// Run acquires one job and processes it
func (w *Worker) Run() error {
	start := time.Now()
	ctx := context.Background()

	job, err := w.jobStore.AcquireJob(ctx)
	if err != nil {
		return err
	}
	if job == nil {
		return nil
	}

	w.logger.Info("Processing job", "jobID", job.ID)

	defer func() {
		duration := fmt.Sprintf("%d ms", time.Since(start).Milliseconds())
		if err != nil {
			w.logger.Warn("Job processing failed", "jobID", job.ID, "reason", errs.ExtractErrorMessage(err), "duration", duration)
		} else {
			w.logger.Info("Job processing completed", "jobID", job.ID, "duration", duration)
		}
	}()

	tmpDir, err := w.createTempDir()
	if err != nil {
		go w.releaseJob(ctx, job)
		return fmt.Errorf("create temp dir for job %s: %w", job.ID, err)
	}
	defer os.RemoveAll(tmpDir)

	originalPath, err := w.downloadOriginal(ctx, job, tmpDir)
	if err != nil {
		if isRetryableError(err) {
			go w.releaseJob(ctx, job)
		} else {
			go w.markJobFailed(ctx, job, errs.ExtractErrorMessage(err))
		}
		return fmt.Errorf("download original for job %s: %w", job.ID, err)
	}

	processedTasks, procErrs := w.processTasks(ctx, job, originalPath)

	if len(procErrs) > 0 {
		err = procErrs[0]
		if isRetryableError(err) {
			go w.releaseJob(ctx, job)
		} else {
			go w.markJobFailed(ctx, job, errs.ExtractErrorMessage(err))
		}
		return fmt.Errorf("job %s: %d tasks failed, first error: %w", job.ID, len(procErrs), err)
	}

	if len(processedTasks) == 0 {
		go w.markJobFailed(ctx, job, "no tasks completed")
		return fmt.Errorf("job %s: no tasks were completed", job.ID)
	}

	if err = w.jobStore.UpdateTasks(ctx, processedTasks); err != nil {
		go w.markJobFailed(ctx, job, errs.ExtractErrorMessage(err))
		return fmt.Errorf("update tasks for job %s: %w", job.ID, err)
	}

	job.MarkCompleted()
	if err = w.jobStore.UpdateJob(ctx, job); err != nil {
		return fmt.Errorf("update job %s to completed: %w", job.ID, err)
	}

	return nil
}

func (w *Worker) createTempDir() (string, error) {
	tmpDir, err := os.MkdirTemp("", "work-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	return tmpDir, nil
}

func (w *Worker) downloadOriginal(ctx context.Context, job *jobs.Job, tmpDir string) (string, error) {
	originalMetadata, err := w.imageStore.GetMetadataByID(ctx, job.OriginalID)
	if err != nil {
		return "", err
	}

	reader, err := w.imageStore.DownloadImage(ctx, job.OriginalID)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	originalPath := filepath.Join(tmpDir, originalMetadata.OriginalName+"."+string(originalMetadata.Format))
	originalFile, err := os.Create(originalPath)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(originalFile, reader)
	if errClose := originalFile.Close(); errClose != nil && err == nil {
		err = errClose
	}

	if err != nil {
		return "", err
	}

	return originalPath, nil
}

func (w *Worker) processTasks(ctx context.Context, job *jobs.Job, originalPath string) ([]*jobs.Task, []error) {
	var wg sync.WaitGroup
	errorsCh := make(chan error, len(job.Tasks))
	resultsCh := make(chan *jobs.Task, len(job.Tasks))
	originalName := fs.BaseNameWithoutExtension(originalPath)

	for _, t := range job.Tasks {
		wg.Add(1)
		go func(task *jobs.Task) {
			defer wg.Done()

			processedPath, err := w.processor.Process(originalPath, task.Options)
			if err != nil {
				errorsCh <- err
				return
			}

			f, err := os.Open(processedPath)
			if err != nil {
				errorsCh <- err
				return
			}
			defer f.Close()

			stats, err := f.Stat()
			if err != nil {
				errorsCh <- err
				return
			}

			variant, err := w.imageStore.UploadImage(ctx, f, originalName, stats.Size())
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

func (w *Worker) markJobFailed(ctx context.Context, job *jobs.Job, reason string) error {
	job.MarkFailed(reason)
	if err := w.jobStore.UpdateJob(ctx, job); err != nil {
		w.logger.Warn("Failed to mark job as failed", "jobID", job.ID, "reason", errs.ExtractErrorMessage(err))
		return err
	}

	return nil
}

func (w *Worker) releaseJob(ctx context.Context, job *jobs.Job) error {
	job.MarkPending()
	if err := w.jobStore.UpdateJob(ctx, job); err != nil {
		w.logger.Warn("Failed to release job", "jobID", job.ID, "reason", errs.ExtractErrorMessage(err))
		return err
	}

	return nil
}

// isRetryableError determines if error is temporary and job should be released for retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for permanent errors that should not be retried
	if errors.Is(err, errs.ErrNotFound) {
		return false
	}
	if errors.Is(err, errs.ErrInvalidInput) {
		return false
	}
	if errors.Is(err, processing.ErrConversionNotSupported) {
		return false
	}

	// All other errors are considered temporary (network, IO, etc.)
	return true
}
