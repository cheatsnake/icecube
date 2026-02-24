package jobstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cheatsnake/icm/internal/domain/image"
	"github.com/cheatsnake/icm/internal/domain/jobs"
	"github.com/cheatsnake/icm/internal/domain/processing"
	sqltool "github.com/cheatsnake/icm/internal/pkg/sql"
)

type jobStorePostgres struct {
	conn *sql.DB
}

func NewJobStorePostgres(conn *sql.DB) (*jobStorePostgres, error) {
	migrations := []sqltool.Migration{
		{
			Version: 1,
			Name:    "init_tables",
			Up: func(tx *sql.Tx) error {
				queries := []string{
					`CREATE TABLE IF NOT EXISTS jobs (
					    id VARCHAR(255) PRIMARY KEY,
					    status VARCHAR(20) NOT NULL,
					    original_id VARCHAR(255) NOT NULL,
					    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					    locked_at TIMESTAMP WITH TIME ZONE,
					    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
					);`,
					`CREATE TABLE IF NOT EXISTS tasks (
					    id VARCHAR(255) PRIMARY KEY,
					    job_id VARCHAR(255) NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
					    variant_id VARCHAR(255),
					    format VARCHAR(10),
					    max_dimension INTEGER,
					    compression_ratio INTEGER,
					    keep_metadata BOOLEAN DEFAULT FALSE,
					    extra JSONB,
					    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
					);`,
					`CREATE INDEX IF NOT EXISTS idx_jobs_pending
					ON jobs (status, created_at)
					WHERE status = 'pending';`,
					`CREATE INDEX IF NOT EXISTS idx_jobs_processing_locked_at
					ON jobs (locked_at)
					WHERE status = 'processing';`,
					`-- Create a trigger to automatically update the updated_at timestamp for jobs
					CREATE OR REPLACE FUNCTION update_jobs_updated_at_column()
					RETURNS TRIGGER AS $$
					BEGIN
					    NEW.updated_at = CURRENT_TIMESTAMP;
					    RETURN NEW;
					END;
					$$ language 'plpgsql';`,
					`CREATE TRIGGER update_jobs_updated_at
					    BEFORE UPDATE ON jobs
					    FOR EACH ROW
					    EXECUTE FUNCTION update_jobs_updated_at_column();`,
					`-- Create a trigger to automatically update the updated_at timestamp for tasks
					CREATE OR REPLACE FUNCTION update_tasks_updated_at_column()
					RETURNS TRIGGER AS $$
					BEGIN
					    NEW.updated_at = CURRENT_TIMESTAMP;
					    RETURN NEW;
					END;
					$$ language 'plpgsql';`,
					`CREATE TRIGGER update_tasks_updated_at
					    BEFORE UPDATE ON tasks
					    FOR EACH ROW
					    EXECUTE FUNCTION update_tasks_updated_at_column();`,
				}

				for _, query := range queries {
					if _, err := tx.Exec(query); err != nil {
						return fmt.Errorf("failed to execute query: %w, query: %s", err, query)
					}
				}
				return nil
			},
		},
	}

	err := sqltool.RunMigrations(conn, nil, migrations)
	if err != nil {
		return nil, err
	}

	return &jobStorePostgres{conn: conn}, nil
}

func (s *jobStorePostgres) CreateJob(ctx context.Context, job *jobs.Job) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	jobQuery := `
		INSERT INTO jobs (id, status, original_id, created_at, locked_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	if _, err = tx.ExecContext(ctx, jobQuery,
		job.ID,
		job.Status,
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

	return tx.Commit()
}

func (s *jobStorePostgres) GetJob(ctx context.Context, id string) (*jobs.Job, error) {
	jobQuery := `
		SELECT id, status, original_id, created_at, locked_at
		FROM jobs
		WHERE id = $1
	`

	var job jobs.Job
	var lockedAt sql.NullTime
	err := s.conn.QueryRowContext(ctx, jobQuery, id).Scan(
		&job.ID,
		&job.Status,
		&job.OriginalID,
		&job.CreatedAt,
		&lockedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if lockedAt.Valid {
		job.LockedAt = &lockedAt.Time
	}

	tasks, err := s.getTasks(ctx, job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks for job %s: %w", id, err)
	}

	job.Tasks = tasks
	return &job, nil
}

func (s *jobStorePostgres) AcquireJob(ctx context.Context) (*jobs.Job, error) {
	tx, err := s.conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
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
	err = tx.QueryRowContext(ctx, acquireJobQuery).Scan(&job.ID, &job.Status, &job.OriginalID, &job.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
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

	if _, err = tx.ExecContext(ctx, lockJobQuery, job.Status, job.LockedAt, job.ID); err != nil {
		return nil, fmt.Errorf("lock job: %w", err)
	}

	tasks, err := s.getTasks(ctx, job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks for job %s: %w", job.ID, err)
	}

	job.Tasks = tasks
	return &job, tx.Commit()
}

func (s *jobStorePostgres) ReleaseJobs(ctx context.Context, lease time.Duration) error {
	query := `
		UPDATE jobs
		SET status = 'pending',
		    locked_at = NULL
		WHERE status = 'processing'
		  AND locked_at < now() - $1::interval
	`

	_, err := s.conn.ExecContext(ctx, query, fmt.Sprintf("%f seconds", lease.Seconds()))
	if err != nil {
		return fmt.Errorf("release jobs: %w", err)
	}

	return nil
}

func (s *jobStorePostgres) UpdateJob(ctx context.Context, job *jobs.Job) error {
	query := `
		UPDATE jobs
		SET status = $2, locked_at = $3
		WHERE id = $1
	`

	var lockedAt any
	if job.LockedAt != nil {
		lockedAt = *job.LockedAt
	} else {
		lockedAt = nil
	}

	result, err := s.conn.ExecContext(ctx, query, job.ID, job.Status, lockedAt)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("job not found: %s", job.ID)
	}

	return nil
}

func (s *jobStorePostgres) DeleteJob(ctx context.Context, id string) error {
	query := `DELETE FROM jobs WHERE id = $1`
	result, err := s.conn.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("job not found: %s", id)
	}

	return nil
}

func (s *jobStorePostgres) UpdateTask(ctx context.Context, task *jobs.Task) error {
	var variantID any
	if task.VariantID != nil {
		variantID = *task.VariantID
	}

	query := `
		UPDATE tasks
		SET variant_id = $2,
		WHERE id = $1
	`

	rows, err := s.conn.QueryContext(ctx, query, task.ID, variantID)
	if err != nil {
		return err
	}
	defer rows.Close()
	return nil
}

func (s *jobStorePostgres) getTasks(ctx context.Context, jobID string) ([]*jobs.Task, error) {
	query := `
		SELECT id, job_id, variant_id, format, max_dimension, compression_ratio, keep_metadata, extra
		FROM tasks
		WHERE job_id = $1
	`

	rows, err := s.conn.QueryContext(ctx, query, jobID)
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
		var compressionRatio sql.NullInt32
		var keepMetadata sql.NullBool
		var extraData []byte

		err := rows.Scan(
			&task.ID,
			&task.JobID,
			&variantID,
			&format,
			&maxDimension,
			&compressionRatio,
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
		if compressionRatio.Valid {
			task.Options.CompressionRatio = int(compressionRatio.Int32)
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

func (s *jobStorePostgres) createTask(
	ctx context.Context,
	tx *sql.Tx,
	task *jobs.Task,
) error {
	query := `
		INSERT INTO tasks (
			id, job_id, variant_id, format,
			max_dimension, compression_ratio,
			keep_metadata, extra
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	var (
		variantID        any
		format           any
		maxDimension     any
		compressionRatio any
		keepMetadata     = false
		extraData        []byte
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
		if o.CompressionRatio > 0 {
			compressionRatio = o.CompressionRatio
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

	_, err := tx.ExecContext(ctx, query,
		task.ID,
		task.JobID,
		variantID,
		format,
		maxDimension,
		compressionRatio,
		keepMetadata,
		extraData,
	)

	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	return nil
}
