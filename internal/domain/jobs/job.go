package jobs

import (
	"errors"
	"time"

	"github.com/cheatsnake/icm/internal/domain/processing"
	"github.com/cheatsnake/icm/internal/pkg/uuid"
)

type Job struct {
	ID         string     `json:"id"`                 // Unique identifier for the job
	Status     JobStatus  `json:"status"`             // Status of the job
	Reason     *string    `json:"reason,omitempty"`   // Message with reason for failure
	OriginalID string     `json:"originalID"`         // ID of the original image variant
	Tasks      []*Task    `json:"tasks"`              // Tasks associated with the job
	CreatedAt  time.Time  `json:"createdAt"`          // Time when the job was created
	LockedAt   *time.Time `json:"lockedAt,omitempty"` // Time when the job was locked (aquired by worker)
}

func NewJob(originalID string) (*Job, error) {
	return &Job{
		ID:         uuid.V7(),
		Status:     JobStatusPending,
		OriginalID: originalID,
		Tasks:      []*Task{},
		CreatedAt:  time.Now(),
		LockedAt:   nil,
	}, nil
}

func (j *Job) AddTask(options *processing.Options) error {
	if j.Status != JobStatusPending {
		return errors.New("cannot add task to non-pending job")
	}

	task, err := NewTask(j.ID, options)
	if err != nil {
		return err
	}

	j.Tasks = append(j.Tasks, task)
	return nil
}

func (j *Job) MarkFailed(reason string) {
	j.Status = JobStatusFailed
	j.LockedAt = nil
	if reason != "" {
		j.Reason = &reason
	}
}

func (j *Job) MarkCompleted() {
	j.Status = JobStatusCompleted
	j.LockedAt = nil
}

func (j *Job) MarkProcessing(lockedAt *time.Time) {
	j.Status = JobStatusProcessing
	j.LockedAt = lockedAt
}

func (j *Job) MarkPending() {
	j.Status = JobStatusPending
	j.LockedAt = nil
}
