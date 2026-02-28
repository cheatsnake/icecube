package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TableExists(ctx context.Context, db *pgxpool.Pool, tableName string) (bool, error) {
	query := `
        SELECT COUNT(*) > 0
        FROM sqlite_master
        WHERE type = 'table' AND name = ?
    `

	var exists bool
	err := db.QueryRow(ctx, query, tableName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}

	return exists, nil
}
