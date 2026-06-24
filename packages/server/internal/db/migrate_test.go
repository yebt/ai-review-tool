package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestMigrateCreatesMVPSchema(t *testing.T) {
	database := openTestDB(t)

	if err := Migrate(context.Background(), database); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	for _, table := range []string{
		"schema_migrations",
		"repos",
		"model_configs",
		"channels",
		"repo_channels",
		"skills",
		"repo_skills",
		"reviews",
		"review_comments",
		"repo_memory",
	} {
		t.Run("table "+table, func(t *testing.T) {
			assertTableExists(t, database, table)
		})
	}

	for _, index := range []string{
		"idx_repos_platform_active",
		"idx_model_configs_repo_active",
		"idx_channels_type_active",
		"idx_repo_channels_channel",
		"idx_skills_dimension_active",
		"idx_repo_skills_skill",
		"idx_reviews_repo_created",
		"idx_reviews_repo_mr",
		"idx_reviews_status",
		"idx_review_comments_review_status",
		"idx_review_comments_dimension_severity",
		"idx_repo_memory_repo_type",
		"idx_repo_memory_repo_dimension",
	} {
		t.Run("index "+index, func(t *testing.T) {
			assertIndexExists(t, database, index)
		})
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	database := openTestDB(t)

	for i := 0; i < 2; i++ {
		if err := Migrate(context.Background(), database); err != nil {
			t.Fatalf("Migrate() run %d error = %v", i+1, err)
		}
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	database, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { database.Close() })

	return database
}

func assertTableExists(t *testing.T, database *sql.DB, table string) {
	t.Helper()

	var name string
	err := database.QueryRow(
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?",
		table,
	).Scan(&name)
	if err != nil {
		t.Fatalf("table %s does not exist: %v", table, err)
	}
}

func assertIndexExists(t *testing.T, database *sql.DB, index string) {
	t.Helper()

	var name string
	err := database.QueryRow(
		"SELECT name FROM sqlite_master WHERE type = 'index' AND name = ?",
		index,
	).Scan(&name)
	if err != nil {
		t.Fatalf("index %s does not exist: %v", index, err)
	}
}
