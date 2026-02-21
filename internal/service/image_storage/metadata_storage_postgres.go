package image_storage

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/cheatsnake/icm/internal/domain/image"
	sqltool "github.com/cheatsnake/icm/internal/pkg/sql"
)

type metadataStoragePostgres struct {
	conn *sql.DB
}

// NewMetadataStoragePostgres creates a new PostgreSQL metadata storage
func NewMetadataStoragePostgres(conn *sql.DB) (MetadataStorage, error) {
	err := sqltool.RunMigrations(conn, nil, migrations)
	if err != nil {
		return nil, err
	}

	return &metadataStoragePostgres{conn: conn}, nil
}

// Get retrieves a single variant by ID
func (s *metadataStoragePostgres) Get(id string) (*image.Variant, error) {
	query := `
		SELECT id, format, width, height, byte_size
		FROM image_variants
		WHERE id = $1
	`

	var variant image.Variant
	err := s.conn.QueryRow(query, id).Scan(
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

// GetMany retrieves multiple variants by their IDs
func (s *metadataStoragePostgres) GetMany(ids []string) ([]*image.Variant, error) {
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

	rows, err := s.conn.Query(query, args...)
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

// Add inserts a new variant into the database
func (s *metadataStoragePostgres) Add(id string, metadata *image.Variant) error {
	query := `
		INSERT INTO image_variants (id, format, width, height, byte_size)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			format = EXCLUDED.format,
			width = EXCLUDED.width,
			height = EXCLUDED.height,
			byte_size = EXCLUDED.byte_size
	`

	_, err := s.conn.Exec(query,
		id,
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

// Delete removes a variant from the database
func (s *metadataStoragePostgres) Delete(id string) error {
	query := `DELETE FROM image_variants WHERE id = $1`

	result, err := s.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete variant: %w", err)
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("variant not found: %s", id)
	}

	return nil
}

// DeleteMany removes multiple variants from the database
func (s *metadataStoragePostgres) DeleteMany(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	tx, err := s.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var errs []error
	for _, id := range ids {
		query := `DELETE FROM image_variants WHERE id = $1`

		_, err := tx.Exec(query, id)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to delete variant %s: %w", id, err))
		}
	}

	if len(errs) == 0 {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
		tx = nil
		return nil
	}

	errorMessages := make([]string, len(errs))
	for i, err := range errs {
		errorMessages[i] = err.Error()
	}
	return fmt.Errorf("failed to delete some variants: %s", strings.Join(errorMessages, "; "))
}
