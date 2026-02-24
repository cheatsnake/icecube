package jobstore

import (
	"context"
	"database/sql"
	"time"

	"github.com/cheatsnake/icm/internal/domain/jobs"
)

type Store struct {
	db *jobStorePostgres
}

func NewStore(conn *sql.DB) *Store {
	return &Store{db: &jobStorePostgres{conn: conn}}
}

func (s *Store) AcquireJob(ctx context.Context) (*jobs.Job, error) {
	return s.db.AcquireJob(ctx)
}

func (s *Store) CreateJob(ctx context.Context, job *jobs.Job) error {
	return s.db.CreateJob(ctx, job)
}

func (s *Store) DeleteJob(ctx context.Context, id string) error {
	return s.db.DeleteJob(ctx, id)
}

func (s *Store) GetJob(ctx context.Context, id string) (*jobs.Job, error) {
	return s.db.GetJob(ctx, id)
}

func (s *Store) ReleaseJobs(ctx context.Context, lease time.Duration) error {
	return s.db.ReleaseJobs(ctx, lease)
}

func (s *Store) UpdateJob(ctx context.Context, job *jobs.Job) error {
	return s.db.UpdateJob(ctx, job)
}

func (s *Store) UpdateTask(ctx context.Context, task *jobs.Task) error {
	return s.db.UpdateTask(ctx, task)
}
