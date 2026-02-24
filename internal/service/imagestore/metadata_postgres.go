package imagestore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/cheatsnake/icm/internal/domain/image"
	sqltool "github.com/cheatsnake/icm/internal/pkg/sql"
)

type metadataStorePostgres struct {
	conn *sql.DB
}

func newMetadataStorePostgres(conn *sql.DB) (*metadataStorePostgres, error) {
	migrations := []sqltool.Migration{
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

	err := sqltool.RunMigrations(conn, nil, migrations)
	if err != nil {
		return nil, err
	}

	return &metadataStorePostgres{conn: conn}, nil
}

func (s *metadataStorePostgres) GetMetadataByID(ctx context.Context, id string) (*image.Variant, error) {
	query := `
		SELECT id, format, width, height, byte_size
		FROM image_variants
		WHERE id = $1
	`

	var variant image.Variant
	err := s.conn.QueryRowContext(ctx, query, id).Scan(
		&variant.ID,
		&variant.Format,
		&variant.Width,
		&variant.Height,
		&variant.ByteSize,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("variant not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get variant: %w", err)
	}

	return &variant, nil
}

func (s *metadataStorePostgres) GetMetadataByIDs(ctx context.Context, ids []string) ([]*image.Variant, error) {
	if len(ids) == 0 {
		return []*image.Variant{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, format, width, height, byte_size
		FROM image_variants
		WHERE id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := s.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query variants: %w", err)
	}
	defer rows.Close()

	variants := make([]*image.Variant, 0, len(ids))
	for rows.Next() {
		var variant image.Variant
		err := rows.Scan(
			&variant.ID,
			&variant.Format,
			&variant.Width,
			&variant.Height,
			&variant.ByteSize,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan variant: %w", err)
		}
		variants = append(variants, &variant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return variants, nil
}

func (s *metadataStorePostgres) AddMetadata(ctx context.Context, metadata *image.Variant) error {
	query := `
		INSERT INTO image_variants (id, format, width, height, byte_size)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			format = EXCLUDED.format,
			width = EXCLUDED.width,
			height = EXCLUDED.height,
			byte_size = EXCLUDED.byte_size
	`

	_, err := s.conn.ExecContext(ctx, query,
		metadata.ID,
		metadata.Format,
		metadata.Width,
		metadata.Height,
		metadata.ByteSize,
	)

	if err != nil {
		return fmt.Errorf("failed to add variant: %w", err)
	}

	return nil
}

func (s *metadataStorePostgres) DeleteMetadataByID(ctx context.Context, id string) error {
	query := `DELETE FROM image_variants WHERE id = $1`

	result, err := s.conn.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete variant: %w", err)
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("variant not found: %s", id)
	}

	return nil
}

func (s *metadataStorePostgres) DeleteMetadataByIDs(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`DELETE FROM image_variants WHERE id IN (%s)`,
		strings.Join(placeholders, ","))

	_, err := s.conn.ExecContext(ctx, query, args...)
	return err
}
