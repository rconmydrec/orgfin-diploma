package database

import (
	"database/sql"
	"log/slog"

	_ "github.com/lib/pq"
)

func New(databaseURL string, log *slog.Logger) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Error("failed to open database", "error", err)
		return nil, err
	}

	if err := db.Ping(); err != nil {
		log.Error("failed to ping database", "error", err)
		db.Close()
		return nil, err
	}

	log.Info("database connection established")
	return db, nil
}
