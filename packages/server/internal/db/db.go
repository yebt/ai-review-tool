package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open creates a SQLite database connection and enables foreign keys.
func Open(path string) (*sql.DB, error) {
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		database.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	return database, nil
}
