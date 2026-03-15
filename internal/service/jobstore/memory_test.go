package jobstore

import (
	"context"
	"testing"
	"time"

	"github.com/cheatsnake/icecube/internal/domain/image"
	"github.com/cheatsnake/icecube/internal/domain/jobs"
	"github.com/cheatsnake/icecube/internal/domain/processing"
)

func TestNewJobStoreMemory(t *testing.T) {
	store := NewJobStoreMemory()
	if store == nil {
		t.Error("NewJobStoreMemory() returned nil")
	}
}

func TestCreateJob(t *testing.T) {
	store := NewJobStoreMemory()
	ctx := context.Background()

	job, _ := jobs.NewJob("original-123")
	opts, _ := processing.NewOptions(image.FormatPNG, 100, 80, false, nil)
	job.AddTask(opts)

	err := store.CreateJob(ctx, job)
	if err != nil {
		t.Errorf("CreateJob() error = %v", err)
	}

	// Duplicate job should fail
	err = store.CreateJob(ctx, job)
	if err == nil {
		t.Error("CreateJob() should return error for duplicate job")
	}
}

func TestGetJob(t *testing.T) {
	store := NewJobStoreMemory()
	ctx := context.Background()

	job, _ := jobs.NewJob("original-123")
	opts, _ := processing.NewOptions(image.FormatPNG, 100, 80, false, nil)
	job.AddTask(opts)
	store.CreateJob(ctx, job)

	// Get existing job
	got, err := store.GetJob(ctx, job.ID)
	if err != nil {
		t.Errorf("GetJob() error = %v", err)
	}
	if got.ID != job.ID {
		t.Errorf("GetJob() ID = %v, want %v", got.ID, job.ID)
	}
	if len(got.Tasks) != 1 {
		t.Errorf("GetJob() Tasks len = %v, want 1", len(got.Tasks))
	}

	// Get non-existing job
	_, err = store.GetJob(ctx, "non-existing")
	if err == nil {
		t.Error("GetJob() should return error for non-existing job")
	}
}

func TestAcquireJob(t *testing.T) {
	store := NewJobStoreMemory()
	ctx := context.Background()

	// No jobs to acquire
	job, err := store.AcquireJob(ctx)
	if err != nil {
		t.Errorf("AcquireJob() error = %v", err)
	}
	if job != nil {
		t.Error("AcquireJob() should return nil when no jobs")
	}

	// Create and acquire job
	j, _ := jobs.NewJob("original-123")
	opts, _ := processing.NewOptions(image.FormatPNG, 100, 80, false, nil)
	j.AddTask(opts)
	store.CreateJob(ctx, j)

	job, err = store.AcquireJob(ctx)
	if err != nil {
		t.Errorf("AcquireJob() error = %v", err)
	}
	if job == nil {
		t.Error("AcquireJob() should return job")
	}
	if job.Status != jobs.JobStatusProcessing {
		t.Errorf("AcquireJob() Status = %v, want %v", job.Status, jobs.JobStatusProcessing)
	}
	if job.LockedAt == nil {
		t.Error("AcquireJob() LockedAt should be set")
	}

	// Second acquire should get next job
	j2, _ := jobs.NewJob("original-456")
	store.CreateJob(ctx, j2)

	job2, err := store.AcquireJob(ctx)
	if err != nil {
		t.Errorf("AcquireJob() error = %v", err)
	}
	if job2 == nil {
		t.Error("AcquireJob() should return second job")
	}
}

func TestReleaseJobs(t *testing.T) {
	store := NewJobStoreMemory()
	ctx := context.Background()

	j, _ := jobs.NewJob("original-123")
	store.CreateJob(ctx, j)

	// Acquire and release
	job, _ := store.AcquireJob(ctx)
	if job.Status != jobs.JobStatusProcessing {
		t.Errorf("Job status = %v, want processing", job.Status)
	}

	// Release with very short lease - should not release recent lock
	err := store.ReleaseJobs(ctx, 1*time.Millisecond)
	if err != nil {
		t.Errorf("ReleaseJobs() error = %v", err)
	}

	// Job should still be processing (not enough time passed)
	got, _ := store.GetJob(ctx, j.ID)
	if got.Status != jobs.JobStatusProcessing {
		t.Errorf("After short ReleaseJobs() Status = %v, want processing", got.Status)
	}

	// Wait and release with longer lease
	time.Sleep(10 * time.Millisecond)
	err = store.ReleaseJobs(ctx, 1*time.Millisecond)
	if err != nil {
		t.Errorf("ReleaseJobs() error = %v", err)
	}

	got, _ = store.GetJob(ctx, j.ID)
	if got.Status != jobs.JobStatusPending {
		t.Errorf("After ReleaseJobs() Status = %v, want pending", got.Status)
	}
}

func TestUpdateJob(t *testing.T) {
	store := NewJobStoreMemory()
	ctx := context.Background()

	j, _ := jobs.NewJob("original-123")
	store.CreateJob(ctx, j)

	// Update job status
	j.Status = jobs.JobStatusCompleted
	err := store.UpdateJob(ctx, j)
	if err != nil {
		t.Errorf("UpdateJob() error = %v", err)
	}

	got, _ := store.GetJob(ctx, j.ID)
	if got.Status != jobs.JobStatusCompleted {
		t.Errorf("GetJob() Status = %v, want completed", got.Status)
	}

	// Update non-existing job
	nonExisting, _ := jobs.NewJob("original-456")
	err = store.UpdateJob(ctx, nonExisting)
	if err == nil {
		t.Error("UpdateJob() should return error for non-existing job")
	}
}

func TestDeleteJob(t *testing.T) {
	store := NewJobStoreMemory()
	ctx := context.Background()

	j, _ := jobs.NewJob("original-123")
	opts, _ := processing.NewOptions(image.FormatPNG, 100, 80, false, nil)
	j.AddTask(opts)
	store.CreateJob(ctx, j)

	// Delete existing job
	err := store.DeleteJob(ctx, j.ID)
	if err != nil {
		t.Errorf("DeleteJob() error = %v", err)
	}

	// Verify deleted
	_, err = store.GetJob(ctx, j.ID)
	if err == nil {
		t.Error("GetJob() should return error after delete")
	}

	// Delete non-existing job
	err = store.DeleteJob(ctx, "non-existing")
	if err == nil {
		t.Error("DeleteJob() should return error for non-existing job")
	}
}

func TestUpdateTask(t *testing.T) {
	store := NewJobStoreMemory()
	ctx := context.Background()

	j, _ := jobs.NewJob("original-123")
	opts, _ := processing.NewOptions(image.FormatPNG, 100, 80, false, nil)
	j.AddTask(opts)
	store.CreateJob(ctx, j)

	task := j.Tasks[0]
	task.Complete("variant-123")

	err := store.UpdateTask(ctx, task)
	if err != nil {
		t.Errorf("UpdateTask() error = %v", err)
	}

	// Verify updated
	got, _ := store.GetJob(ctx, j.ID)
	if got.Tasks[0].VariantID == nil || *got.Tasks[0].VariantID != "variant-123" {
		t.Error("Task should have variantID after update")
	}
}

func TestUpdateTasks(t *testing.T) {
	store := NewJobStoreMemory()
	ctx := context.Background()

	j, _ := jobs.NewJob("original-123")
	opts1, _ := processing.NewOptions(image.FormatPNG, 100, 80, false, nil)
	opts2, _ := processing.NewOptions(image.FormatJPEG, 200, 90, false, nil)
	j.AddTask(opts1)
	j.AddTask(opts2)
	store.CreateJob(ctx, j)

	// Complete both tasks
	j.Tasks[0].Complete("variant-1")
	j.Tasks[1].Complete("variant-2")

	err := store.UpdateTasks(ctx, j.Tasks)
	if err != nil {
		t.Errorf("UpdateTasks() error = %v", err)
	}

	got, _ := store.GetJob(ctx, j.ID)
	if got.Tasks[0].VariantID == nil || got.Tasks[1].VariantID == nil {
		t.Error("All tasks should have variantID after update")
	}
}

func TestUpdateTasks_EmptyList(t *testing.T) {
	store := NewJobStoreMemory()
	ctx := context.Background()

	err := store.UpdateTasks(ctx, []*jobs.Task{})
	if err != nil {
		t.Errorf("UpdateTasks() error = %v", err)
	}
}
