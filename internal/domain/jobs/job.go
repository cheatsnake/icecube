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
