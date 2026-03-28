package imagestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/cheatsnake/icecube/internal/domain/errs"
	"github.com/cheatsnake/icecube/internal/domain/image"
	"github.com/jackc/pgx/v5/pgxpool"
)

type metadataStorePostgres struct {
	logger *slog.Logger
	conn   *pgxpool.Pool
}

func newMetadataStorePostgres(logger *slog.Logger, conn *pgxpool.Pool) *metadataStorePostgres {
	return &metadataStorePostgres{logger: logger, conn: conn}
}

func (s *metadataStorePostgres) GetMetadataByID(ctx context.Context, id string) (*image.Variant, error) {
	query := `
		SELECT id, original_name, format, width, height, byte_size
		FROM image_metadata
		WHERE id = $1
	`

	var variant image.Variant
	err := s.conn.QueryRow(ctx, query, id).Scan(
		&variant.ID,
		&variant.OriginalName,
		&variant.Format,
		&variant.Width,
		&variant.Height,
		&variant.ByteSize,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Join(errs.ErrNotFound, errors.New("variant not found: "+id))
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
		SELECT id, original_name, format, width, height, byte_size
		FROM image_metadata
		WHERE id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query variants: %w", err)
	}
	defer rows.Close()

	variants := make([]*image.Variant, 0, len(ids))
	for rows.Next() {
		var variant image.Variant
		err := rows.Scan(
			&variant.ID,
			&variant.OriginalName,
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
		INSERT INTO image_metadata (id, original_name, format, width, height, byte_size)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			original_name = EXCLUDED.original_name,
			format = EXCLUDED.format,
			width = EXCLUDED.width,
			height = EXCLUDED.height,
			byte_size = EXCLUDED.byte_size
	`

	_, err := s.conn.Exec(ctx, query,
		metadata.ID,
		metadata.OriginalName,
		metadata.Format,
		metadata.Width,
		metadata.Height,
		metadata.ByteSize,
	)

	if err != nil {
		return fmt.Errorf("failed to add variant: %w", err)
	}

	s.logger.Debug("Metadata added", "id", metadata.ID, "format", metadata.Format)
	return nil
}

func (s *metadataStorePostgres) DeleteMetadataByID(ctx context.Context, id string) error {
	query := `DELETE FROM image_metadata WHERE id = $1`

	result, err := s.conn.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete variant: %w", err)
	}

	if rows := result.RowsAffected(); rows == 0 {
		return errors.Join(errs.ErrNotFound, errors.New("variant not found: "+id))
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

	query := fmt.Sprintf(`DELETE FROM image_metadata WHERE id IN (%s)`,
		strings.Join(placeholders, ","))

	_, err := s.conn.Exec(ctx, query, args...)
	return err
}
