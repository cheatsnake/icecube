package jobstore

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/cheatsnake/icecube/internal/domain/jobs"
	"github.com/jackc/pgx/v5/pgxpool"
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
	CountPendingJobs(ctx context.Context) (int, error)
	SubscribeOnJob() chan struct{}
	UnsubscribeOnJob(ch chan struct{})
}

type Config struct {
	Type string
}

func New(cfg Config, pool *pgxpool.Pool, logger *slog.Logger) (Store, error) {
	switch cfg.Type {
	case "memory":
		return newJobStoreMemory(logger), nil
	case "postgres":
		return newJobStorePostgres(pool, logger), nil
	default:
		return nil, errors.New("unsupported jobstore type: " + cfg.Type)
	}
}

func newJobStoreMemory(logger *slog.Logger) *jobStoreMemory {
	return &jobStoreMemory{
		logger:      logger,
		jobs:        make(map[string]*jobs.Job),
		tasks:       make(map[string]*jobs.Task),
		notifyCh:    make(chan struct{}, 1),
		subscribers: make([]chan struct{}, 0),
	}
}

func newJobStorePostgres(pool *pgxpool.Pool, logger *slog.Logger) *jobStorePostgres {
	notifyCh := make(chan struct{}, 1)
	listenCh := make(chan struct{}, 1)

	store := &jobStorePostgres{
		conn:        pool,
		notifyCh:    notifyCh,
		subscribers: make([]chan struct{}, 0),
		listenCh:    listenCh,
		listenDone:  make(chan struct{}),
		logger:      logger,
	}

	go store.startListener()

	return store
}
