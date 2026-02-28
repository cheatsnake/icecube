package postgres

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(dsn string, log *slog.Logger) (*pgxpool.Pool, error) {
	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	err = runMigrations(ctx, db, log, migrations)
	if err != nil {
		return nil, err
	}

	return db, nil
}
