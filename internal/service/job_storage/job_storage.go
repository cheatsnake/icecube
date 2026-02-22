package job_storage

import (
	"context"
	"time"

	"github.com/cheatsnake/icm/internal/domain/jobs"
)

type Storage interface {
	CreateJob(ctx context.Context, job *jobs.Job) error
	GetJob(ctx context.Context, id string) (*jobs.Job, error)
	AcquireJob(ctx context.Context) (*jobs.Job, error)
	ReleaseJobs(ctx context.Context, lease time.Duration) error
	UpdateJob(ctx context.Context, job *jobs.Job) error
	DeleteJob(ctx context.Context, id string) error
	UpdateTask(ctx context.Context, task *jobs.Task) error
}
