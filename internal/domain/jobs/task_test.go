package jobs

import (
	"testing"

	"github.com/cheatsnake/icm/internal/domain/image"
	"github.com/cheatsnake/icm/internal/domain/processing"
)

func TestTask_Complete(t *testing.T) {
	opts, _ := processing.NewOptions(image.FormatPNG, 100, 80, false, nil)
	job, _ := NewJob("original-123")
	job.AddTask(opts)

	task := job.Tasks[0]
	if task.VariantID != nil {
		t.Error("Task should not have VariantID before completion")
	}

	variantID := "variant-456"
	task.Complete(variantID)

	if task.VariantID == nil {
		t.Error("Task should have VariantID after completion")
	}
	if *task.VariantID != variantID {
		t.Errorf("Task.VariantID = %v, want %v", *task.VariantID, variantID)
	}
}
