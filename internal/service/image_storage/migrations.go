package image_storage

import (
	"database/sql"
	"fmt"

	sqltool "github.com/cheatsnake/icm/internal/pkg/sql"
)

var migrations = []sqltool.Migration{
	{
		Version: 1,
		Name:    "init_table",
		Up: func(tx *sql.Tx) error {
			queries := []string{
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
				if _, err := tx.Exec(query); err != nil {
					return fmt.Errorf("failed to execute query: %w, query: %s", err, query)
				}
			}
			return nil
		},
	},
}
