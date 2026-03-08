package jobs

import (
	"github.com/cheatsnake/icm/internal/domain/processing"
	"github.com/cheatsnake/icm/internal/pkg/uuid"
)

type Task struct {
	ID        string              `json:"id"`
	JobID     string              `json:"-"`
	Options   *processing.Options `json:"options"`
	VariantID *string             `json:"variantID"`
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
