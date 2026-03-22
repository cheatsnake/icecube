package jobstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/cheatsnake/icecube/internal/domain/errs"
	"github.com/cheatsnake/icecube/internal/domain/image"
	"github.com/cheatsnake/icecube/internal/domain/jobs"
	"github.com/cheatsnake/icecube/internal/domain/processing"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type JobStorePostgres struct {
	conn        *pgxpool.Pool
	subscribers []chan struct{}
	notifyCh    chan struct{}
	mu          sync.RWMutex

	listener   *pgxpool.Conn
	listenCh   chan struct{}
	listenDone chan struct{}
	logger     *slog.Logger
}

func NewJobStorePostgres(conn *pgxpool.Pool, logger *slog.Logger) *JobStorePostgres {
	notifyCh := make(chan struct{}, 1)
	listenCh := make(chan struct{}, 1)

	store := &JobStorePostgres{
		conn:        conn,
		notifyCh:    notifyCh,
		subscribers: make([]chan struct{}, 0),
		listenCh:    listenCh,
		listenDone:  make(chan struct{}),
		logger:      logger,
	}

	go store.startListener()

	return store
}

// Close releases PostgreSQL listener resources
func (s *JobStorePostgres) Close() error {
	close(s.listenDone)
	if s.listener != nil {
		s.listener.Release()
	}
	return nil
}

func (s *JobStorePostgres) CreateJob(ctx context.Context, job *jobs.Job) error {
	tx, err := s.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	jobQuery := `
		INSERT INTO jobs (id, status, reason, original_id, created_at, locked_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	if _, err = tx.Exec(ctx, jobQuery,
		job.ID,
		job.Status,
		job.Reason,
		job.OriginalID,
		job.CreatedAt,
		job.LockedAt,
	); err != nil {
		return fmt.Errorf("create job: %w", err)
	}

	for _, task := range job.Tasks {
		if err = s.createTask(ctx, tx, task); err != nil {
			return fmt.Errorf("create task for job %s: %w", job.ID, err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	s.logger.Debug("Job created", "jobID", job.ID, "taskCount", len(job.Tasks))
	s.notifySubscribers()
	return nil
}

func (s *JobStorePostgres) GetJob(ctx context.Context, id string) (*jobs.Job, error) {
	jobQuery := `
		SELECT id, status, reason, original_id, created_at, locked_at
		FROM jobs
		WHERE id = $1
	`

	var job jobs.Job
	var lockedAt sql.NullTime
	var reason sql.NullString
	err := s.conn.QueryRow(ctx, jobQuery, id).Scan(
		&job.ID,
		&job.Status,
		&reason,
		&job.OriginalID,
		&job.CreatedAt,
		&lockedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Join(errs.ErrNotFound, errors.New("job not found: "+id))
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if lockedAt.Valid {
		job.LockedAt = &lockedAt.Time
	}

	if reason.Valid {
		job.Reason = &reason.String
	}

	tasks, err := s.getTasks(ctx, job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks for job %s: %w", id, err)
	}

	job.Tasks = tasks
	return &job, nil
}

func (s *JobStorePostgres) AcquireJob(ctx context.Context) (*jobs.Job, error) {
	tx, err := s.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	acquireJobQuery := `
		SELECT id, status, original_id, created_at
		FROM jobs
		WHERE status = 'pending'
		ORDER BY created_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`

	var job jobs.Job
	err = tx.QueryRow(ctx, acquireJobQuery).Scan(&job.ID, &job.Status, &job.OriginalID, &job.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows || strings.Contains(err.Error(), "no rows") {
			s.logger.Debug("No pending jobs available")
			return nil, nil
		}
		return nil, fmt.Errorf("acquire job select: %w", err)
	}

	job.Status = jobs.JobStatusProcessing
	now := time.Now().UTC()
	job.LockedAt = &now

	lockJobQuery := `
		UPDATE jobs
		SET status = $1, locked_at = $2
		WHERE id = $3
	`

	if _, err = tx.Exec(ctx, lockJobQuery, job.Status, job.LockedAt, job.ID); err != nil {
		return nil, fmt.Errorf("lock job: %w", err)
	}

	tasks, err := s.getTasks(ctx, job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks for job %s: %w", job.ID, err)
	}

	job.Tasks = tasks
	s.logger.Debug("Job acquired", "jobID", job.ID, "originalID", job.OriginalID)
	return &job, tx.Commit(ctx)
}

func (s *JobStorePostgres) ReleaseJobs(ctx context.Context, lease time.Duration) error {
	query := `
		UPDATE jobs
		SET status = 'pending',
		    locked_at = NULL
		WHERE status = 'processing'
		  AND locked_at < now() - $1::interval
	`

	_, err := s.conn.Exec(ctx, query, fmt.Sprintf("%f seconds", lease.Seconds()))
	if err != nil {
		return fmt.Errorf("release jobs: %w", err)
	}

	return nil
}

func (s *JobStorePostgres) UpdateJob(ctx context.Context, job *jobs.Job) error {
	query := `
		UPDATE jobs
		SET status = $2, locked_at = $3, reason = $4
		WHERE id = $1
	`

	var lockedAt any
	if job.LockedAt != nil {
		lockedAt = *job.LockedAt
	} else {
		lockedAt = nil
	}

	var reason any
	if job.Reason != nil {
		reason = *job.Reason
	} else {
		reason = nil
	}

	result, err := s.conn.Exec(ctx, query, job.ID, job.Status, lockedAt, reason)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	if rows := result.RowsAffected(); rows == 0 {
		s.logger.Error("Job not found for update", "jobID", job.ID)
		return errors.Join(errs.ErrNotFound, errors.New("job not found: "+job.ID))
	}

	s.logger.Debug("Job updated", "jobID", job.ID, "status", job.Status)
	return nil
}

func (s *JobStorePostgres) DeleteJob(ctx context.Context, id string) error {
	query := `DELETE FROM jobs WHERE id = $1`
	result, err := s.conn.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	if rows := result.RowsAffected(); rows == 0 {
		s.logger.Error("Job not found for deletion", "jobID", id)
		return errors.Join(errs.ErrNotFound, errors.New("job not found: "+id))
	}

	s.logger.Debug("Job deleted", "jobID", id)
	return nil
}

func (s *JobStorePostgres) UpdateTask(ctx context.Context, task *jobs.Task) error {
	var variantID any
	if task.VariantID != nil {
		variantID = *task.VariantID
	}

	query := `UPDATE tasks SET variant_id = $2 WHERE id = $1`
	rows, err := s.conn.Query(ctx, query, task.ID, variantID)
	if err != nil {
		return err
	}

	rows.Close()
	return nil
}

func (s *JobStorePostgres) UpdateTasks(ctx context.Context, tasks []*jobs.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	tx, err := s.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	const query = `UPDATE tasks SET variant_id = $2 WHERE id = $1`

	for _, task := range tasks {
		var variantID any
		if task.VariantID != nil {
			variantID = *task.VariantID
		}

		res, err := tx.Exec(ctx, query, task.ID, variantID)
		if err != nil {
			return err
		}

		if res.RowsAffected() == 0 {
			s.logger.Error("Task not found for update", "taskID", task.ID)
			return sql.ErrNoRows
		}
	}

	s.logger.Debug("Tasks updated", "count", len(tasks))
	return tx.Commit(ctx)
}

func (s *JobStorePostgres) CountPendingJobs(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM jobs WHERE status = 'pending'`
	var count int
	err := s.conn.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count pending jobs: %w", err)
	}
	return count, nil
}

func (s *JobStorePostgres) SubscribeOnJob() chan struct{} {
	ch := make(chan struct{}, 1)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers = append(s.subscribers, ch)
	return ch
}

func (s *JobStorePostgres) UnsubscribeOnJob(ch chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, sub := range s.subscribers {
		if sub == ch {
			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
			close(ch)
			return
		}
	}
}

// startListener connects to PostgreSQL and listens for job notifications
func (s *JobStorePostgres) startListener() {
	ctx := context.Background()

	for {
		select {
		case <-s.listenDone:
			return
		default:
		}

		listenerConn, err := s.conn.Acquire(ctx)
		if err != nil {
			s.logger.Debug("Failed to acquire connection for listener", "error", err)
			time.Sleep(time.Second)
			continue
		}

		s.listener = listenerConn

		_, err = listenerConn.Conn().Exec(ctx, "LISTEN jobs_pending")
		if err != nil {
			s.logger.Warn("Failed to LISTEN on jobs_pending", "error", err)
			listenerConn.Release()
			time.Sleep(time.Second)
			continue
		}

		s.logger.Debug("PostgreSQL pending jobs listener started")

		for {
			notification, err := listenerConn.Conn().WaitForNotification(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				s.logger.Warn("Listener error, reconnecting", "error", err)
				listenerConn.Release()
				break
			}

			s.logger.Debug("Received PostgreSQL notification", "channel", notification.Channel, "payload", notification.Payload)
			s.notifySubscribers()
		}

		time.Sleep(1 * time.Second) // Brief delay before reconnecting
	}
}

func (s *JobStorePostgres) getTasks(ctx context.Context, jobID string) ([]*jobs.Task, error) {
	query := `
		SELECT id, job_id, variant_id, format, max_dimension, quality, keep_metadata, extra
		FROM tasks
		WHERE job_id = $1
	`

	rows, err := s.conn.Query(ctx, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]*jobs.Task, 0)
	for rows.Next() {
		var task jobs.Task
		var variantID sql.NullString
		var format sql.NullString
		var maxDimension sql.NullInt32
		var quality sql.NullInt32
		var keepMetadata sql.NullBool
		var extraData []byte

		err := rows.Scan(
			&task.ID,
			&task.JobID,
			&variantID,
			&format,
			&maxDimension,
			&quality,
			&keepMetadata,
			&extraData,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		if variantID.Valid {
			task.VariantID = &variantID.String
		}

		task.Options = &processing.Options{}
		if format.Valid {
			task.Options.Format = image.Format(format.String)
		}
		if maxDimension.Valid {
			task.Options.MaxDimension = int(maxDimension.Int32)
		}
		if quality.Valid {
			task.Options.Quality = int(quality.Int32)
		}
		if keepMetadata.Valid {
			task.Options.KeepMetadata = keepMetadata.Bool
		}

		if len(extraData) > 0 {
			var extraMap map[string]string
			if err := json.Unmarshal(extraData, &extraMap); err == nil {
				task.Options.Extra = extraMap
			}
		}

		tasks = append(tasks, &task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return tasks, nil
}

func (s *JobStorePostgres) createTask(
	ctx context.Context,
	tx pgx.Tx,
	task *jobs.Task,
) error {
	query := `
		INSERT INTO tasks (
			id, job_id, variant_id, format,
			max_dimension, quality,
			keep_metadata, extra
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	var (
		variantID    any
		format       any
		maxDimension any
		quality      any
		keepMetadata = false
		extraData    []byte
	)

	if task.VariantID != nil {
		variantID = *task.VariantID
	}

	if o := task.Options; o != nil {
		if o.Format != "" {
			format = o.Format
		}
		if o.MaxDimension > 0 {
			maxDimension = o.MaxDimension
		}
		if o.Quality > 0 {
			quality = o.Quality
		}
		keepMetadata = o.KeepMetadata

		if len(o.Extra) > 0 {
			var err error
			extraData, err = json.Marshal(o.Extra)
			if err != nil {
				return fmt.Errorf("marshal extra: %w", err)
			}
		}
	}

	_, err := tx.Exec(ctx, query,
		task.ID,
		task.JobID,
		variantID,
		format,
		maxDimension,
		quality,
		keepMetadata,
		extraData,
	)

	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	return nil
}

func (s *JobStorePostgres) notifySubscribers() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.logger.Debug("Notifying subscribers", "count", len(s.subscribers))
	for _, ch := range s.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
