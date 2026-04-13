package database

import (
	"database/sql"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Connect(databaseURL string) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for i := 0; i < 5; i++ {
		db, err = sql.Open("pgx", databaseURL)
		if err != nil {
			slog.Warn("failed to open database", "attempt", i+1, "error", err)
			time.Sleep(time.Second)
			continue
		}

		if err = db.Ping(); err != nil {
			slog.Warn("failed to ping database", "attempt", i+1, "error", err)
			db.Close()
			time.Sleep(time.Second)
			continue
		}

		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)

		slog.Info("database connected")
		return db, nil
	}

	return nil, err
}
