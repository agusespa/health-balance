package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func Init(dbPath string) (*DB, error) {
	dsn := dbPath + "?_journal=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	if err := createTables(db); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS user_profile (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			birth_date TEXT NOT NULL,
			sex TEXT NOT NULL,
			height_cm REAL NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS daily_metrics (
			date TEXT NOT NULL,
			metric TEXT NOT NULL,
			value REAL NOT NULL,
			samples INTEGER NOT NULL DEFAULT 0,
			source TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL,
			PRIMARY KEY (date, metric)
		) WITHOUT ROWID;`,
		`CREATE INDEX IF NOT EXISTS idx_daily_metrics_metric_date ON daily_metrics (metric, date);`,
		`CREATE TABLE IF NOT EXISTS ingest_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			received_at TEXT NOT NULL,
			kind TEXT NOT NULL,
			rows INTEGER NOT NULL,
			accepted INTEGER NOT NULL,
			rejected INTEGER NOT NULL,
			from_date TEXT NOT NULL DEFAULT '',
			to_date TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE INDEX IF NOT EXISTS idx_ingest_runs_kind_received ON ingest_runs (kind, received_at);`,
		`CREATE TABLE IF NOT EXISTS insights (
			week_end TEXT PRIMARY KEY,
			model TEXT NOT NULL,
			created_at TEXT NOT NULL,
			markdown TEXT NOT NULL
		);`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}
