package sql

import (
	"database/sql"
	"fmt"
)

func ColumnExists(db *sql.DB, tableName, columnName string) (bool, error) {
	query := `
        SELECT COUNT(*) > 0
        FROM pragma_table_info(?)
        WHERE name = ?
    `

	var exists bool
	err := db.QueryRow(query, tableName, columnName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check column existence: %w", err)
	}

	return exists, nil
}

func TableExists(db *sql.DB, tableName string) (bool, error) {
	query := `
        SELECT COUNT(*) > 0
        FROM sqlite_master
        WHERE type = 'table' AND name = ?
    `

	var exists bool
	err := db.QueryRow(query, tableName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}

	return exists, nil
}
