package bootstrap

import (
	"context"
	"database/sql"
	"fmt"

	"co-review/server/internal/config"
	"co-review/server/internal/db"
)

type ErrorReport struct {
	Title string
	Type  string
	Err   error
}

func (e ErrorReport) Error() string {
	return fmt.Sprintf("%s: %v", e.Title, e.Err)
}

func (e ErrorReport) Unwrap() error {
	return e.Err
}

// Init bootstraps the application by loading configuration, opening the
// database and running migrations. The caller is responsible for closing the
// returned *sql.DB.
func Init() (cfg config.Config, database *sql.DB, err error) {
	cfg, err = config.Load()
	if err != nil {
		return config.Config{}, nil, ErrorReport{Title: "load configuration", Type: "error", Err: err}
	}

	database, err = db.Open(cfg.DatabaseURL)
	if err != nil {
		return config.Config{}, nil, ErrorReport{Title: "open database", Type: "error", Err: err}
	}

	if err := db.Migrate(context.Background(), database); err != nil {
		database.Close()
		return config.Config{}, nil, ErrorReport{Title: "run migrations", Type: "error", Err: err}
	}

	return cfg, database, nil
}