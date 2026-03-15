package jobs

import (
	"testing"
	"time"

	"github.com/cheatsnake/icecube/internal/domain/image"
	"github.com/cheatsnake/icecube/internal/domain/processing"
)

func TestNewJob(t *testing.T) {
	job, err := NewJob("original-123")
	if err != nil {
		t.Fatalf("NewJob() error = %v", err)
	}

	if job.ID == "" {
		t.Error("NewJob() ID should not be empty")
	}
	if job.Status != JobStatusPending {
		t.Errorf("NewJob() Status = %v, want %v", job.Status, JobStatusPending)
	}
	if job.OriginalID != "original-123" {
		t.Errorf("NewJob() OriginalID = %v, want original-123", job.OriginalID)
	}
	if len(job.Tasks) != 0 {
		t.Errorf("NewJob() Tasks len = %d, want 0", len(job.Tasks))
	}
	if job.CreatedAt.IsZero() {
		t.Error("NewJob() CreatedAt should not be zero")
	}
	if job.LockedAt != nil {
		t.Error("NewJob() LockedAt should be nil")
	}
}

func TestJob_AddTask(t *testing.T) {
	job, _ := NewJob("original-123")

	opts, _ := processing.NewOptions(image.FormatPNG, 100, 80, false, nil)

	// Add task to pending job - should succeed
	err := job.AddTask(opts)
	if err != nil {
		t.Errorf("AddTask() on pending job error = %v", err)
	}
	if len(job.Tasks) != 1 {
		t.Errorf("AddTask() Tasks len = %v, want 1", len(job.Tasks))
	}

	// Add another task
	err = job.AddTask(opts)
	if err != nil {
		t.Errorf("AddTask() second task error = %v", err)
	}
	if len(job.Tasks) != 2 {
		t.Errorf("AddTask() Tasks len = %v, want 2", len(job.Tasks))
	}

	// Add task to processing job - should fail
	job.MarkProcessing(nil)
	err = job.AddTask(opts)
	if err == nil {
		t.Error("AddTask() on processing job should return error")
	}
}

func TestJob_MarkFailed(t *testing.T) {
	job, _ := NewJob("original-123")

	job.MarkFailed("something went wrong")

	if job.Status != JobStatusFailed {
		t.Errorf("MarkFailed() Status = %v, want %v", job.Status, JobStatusFailed)
	}
	if job.LockedAt != nil {
		t.Error("MarkFailed() LockedAt should be nil")
	}
	if job.Reason == nil || *job.Reason != "something went wrong" {
		t.Errorf("MarkFailed() Reason = %v, want 'something went wrong'", job.Reason)
	}
}

func TestJob_MarkFailed_EmptyReason(t *testing.T) {
	job, _ := NewJob("original-123")

	job.MarkFailed("")

	if job.Status != JobStatusFailed {
		t.Errorf("MarkFailed('') Status = %v, want %v", job.Status, JobStatusFailed)
	}
	if job.Reason != nil {
		t.Error("MarkFailed('') Reason should be nil")
	}
}

func TestJob_MarkCompleted(t *testing.T) {
	job, _ := NewJob("original-123")

	job.MarkCompleted()

	if job.Status != JobStatusCompleted {
		t.Errorf("MarkCompleted() Status = %v, want %v", job.Status, JobStatusCompleted)
	}
	if job.LockedAt != nil {
		t.Error("MarkCompleted() LockedAt should be nil")
	}
}

func TestJob_MarkProcessing(t *testing.T) {
	job, _ := NewJob("original-123")
	now := time.Now()

	job.MarkProcessing(&now)

	if job.Status != JobStatusProcessing {
		t.Errorf("MarkProcessing() Status = %v, want %v", job.Status, JobStatusProcessing)
	}
	if job.LockedAt == nil {
		t.Error("MarkProcessing() LockedAt should not be nil")
	}
}

func TestJob_MarkPending(t *testing.T) {
	job, _ := NewJob("original-123")
	job.MarkProcessing(nil)

	job.MarkPending()

	if job.Status != JobStatusPending {
		t.Errorf("MarkPending() Status = %v, want %v", job.Status, JobStatusPending)
	}
	if job.LockedAt != nil {
		t.Error("MarkPending() LockedAt should be nil")
	}
}
