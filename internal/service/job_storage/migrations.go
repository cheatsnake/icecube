package job_storage

import (
	"database/sql"
	"fmt"

	sqltool "github.com/cheatsnake/icm/internal/pkg/sql"
)

var migrations = []sqltool.Migration{
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
