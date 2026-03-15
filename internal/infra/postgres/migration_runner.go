package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type migration struct {
	Version int
	Name    string
	Up      func(ctx context.Context, tx pgx.Tx) error
	Down    func(ctx context.Context, tx pgx.Tx) error
}

func runMigrations(ctx context.Context, db *pgxpool.Pool, log *slog.Logger, migrations []migration) error {
	// Create migrations table if it doesn't exist
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	var currentVersion int
	err = db.QueryRow(ctx, "SELECT COALESCE(MAX(version), 0) FROM migrations").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	if currentVersion == 0 {
		log.Info("Fresh database, starting migrations from the beginning")
	}

	for _, m := range migrations {
		if m.Version > currentVersion {
			log.Info("Applying migration", "version", m.Version, "name", m.Name)

			tx, err := db.Begin(ctx)
			if err != nil {
				return fmt.Errorf("failed to begin transaction for migration %d: %w", m.Version, err)
			}

			defer func() {
				if tx != nil {
					tx.Rollback(ctx)
				}
			}()

			if err := m.Up(ctx, tx); err != nil {
				return fmt.Errorf("migration %d (%s) failed: %w", m.Version, m.Name, err)
			}

			_, err = tx.Exec(ctx, "INSERT INTO migrations (version, name) VALUES ($1, $2)", m.Version, m.Name)
			if err != nil {
				return fmt.Errorf("failed to record migration %d: %w", m.Version, err)
			}

			if err := tx.Commit(ctx); err != nil {
				return fmt.Errorf("failed to commit migration %d: %w", m.Version, err)
			}

			log.Info("Migration applied successfully", "version", m.Version, "name", m.Name)
		}
	}

	var finalVersion int
	err = db.QueryRow(ctx, "SELECT COALESCE(MAX(version), 0) FROM migrations").Scan(&finalVersion)
	if err != nil {
		return fmt.Errorf("failed to get final version: %w", err)
	}

	if finalVersion > currentVersion {
		log.Info("Database migration completed",
			"from_version", currentVersion,
			"to_version", finalVersion,
			"migrations_applied", finalVersion-currentVersion)
	} else {
		log.Info("Database is up to date", "version", finalVersion)
	}

	return nil
}
