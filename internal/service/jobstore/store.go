package jobstore

import (
	"context"
	"time"

	"github.com/cheatsnake/icecube/internal/domain/jobs"
)

type Store interface {
	CreateJob(ctx context.Context, job *jobs.Job) error
	GetJob(ctx context.Context, id string) (*jobs.Job, error)
	AcquireJob(ctx context.Context) (*jobs.Job, error)
	ReleaseJobs(ctx context.Context, lease time.Duration) error
	UpdateJob(ctx context.Context, job *jobs.Job) error
	DeleteJob(ctx context.Context, id string) error
	UpdateTask(ctx context.Context, task *jobs.Task) error
	UpdateTasks(ctx context.Context, tasks []*jobs.Task) error
}
