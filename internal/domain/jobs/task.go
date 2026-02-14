package jobs

import (
	"github.com/cheatsnake/icm/internal/domain/processing"
	"github.com/cheatsnake/icm/internal/pkg/uuid"
)

type Task struct {
	ID        string
	JobID     string
	Options   *processing.Options
	VariantID *string
}

type TaskStorage interface {
	Create(task *Task) error
	Get(id string) (*Task, error)
	GetMany(ids []string) ([]*Task, error)
	Update(task *Task) error
	Delete(id string) error
}

func NewTask(jobID string, options *processing.Options) (*Task, error) {
	return &Task{
		ID:        uuid.V7(),
		JobID:     jobID,
		Options:   options,
		VariantID: nil,
	}, nil
}

// Complete marks the task as completed by setting the corresponding variant ID
func (t *Task) Complete(variantID string) {
	t.VariantID = &variantID
}
