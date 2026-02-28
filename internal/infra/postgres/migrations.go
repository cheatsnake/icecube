package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var migrations = []migration{
	{
		Version: 1,
		Name:    "init_database",
		Up: func(ctx context.Context, tx pgx.Tx) error {
			queries := []string{
				// Jobs and tasks
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

				`CREATE OR REPLACE FUNCTION notify_pending_jobs()
					RETURNS trigger AS $$
					BEGIN
					  -- INSERT case
					  IF TG_OP = 'INSERT' AND NEW.status = 'pending' AND NEW.locked_at IS NULL
					  THEN
					    PERFORM pg_notify('jobs_pending', NEW.id);
					  END IF;
					  IF TG_OP = 'UPDATE'
					     AND NEW.status = 'pending'
					     AND NEW.locked_at IS NULL
					     AND (
					       OLD.status IS DISTINCT FROM NEW.status
					       OR OLD.locked_at IS DISTINCT FROM NEW.locked_at
					     )
					  THEN
					    PERFORM pg_notify('jobs_pending', NEW.id);
					  END IF;

					  RETURN NEW;
					END;
				$$ LANGUAGE plpgsql;`,

				`CREATE TRIGGER jobs_pending_notify
				AFTER INSERT OR UPDATE
				ON jobs
				FOR EACH ROW
				EXECUTE FUNCTION notify_pending_jobs();`,

				// Image metadata
				`CREATE TABLE IF NOT EXISTS image_metadata (
				    id VARCHAR(255) PRIMARY KEY,
				    format VARCHAR(10) NOT NULL,
				    width INTEGER NOT NULL CHECK (width > 0),
				    height INTEGER NOT NULL CHECK (height > 0),
				    byte_size BIGINT NOT NULL CHECK (byte_size > 0),
				    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
				    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);`,
				`-- Create a trigger to automatically update the updated_at timestamp
				CREATE OR REPLACE FUNCTION update_updated_at_column()
				RETURNS TRIGGER AS $$
				BEGIN
				    NEW.updated_at = CURRENT_TIMESTAMP;
				    RETURN NEW;
				END;
				$$ language 'plpgsql';`,
				`CREATE TRIGGER update_image_metadata_updated_at
				    BEFORE UPDATE ON image_metadata
				    FOR EACH ROW
				    EXECUTE FUNCTION update_updated_at_column();`,
			}

			for _, query := range queries {
				if _, err := tx.Exec(ctx, query); err != nil {
					return fmt.Errorf("failed to execute query: %w, query: %s", err, query)
				}
			}
			return nil
		},
	},
}
