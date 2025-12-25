package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func Init(dbPath string) (*sql.DB, error) {
	// Enable WAL mode and set a busy timeout for better concurrency
	dsn := dbPath + "?_journal=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// Standard connection pool settings for SQLite
	db.SetMaxOpenConns(1)

	if err := createTables(db); err != nil {
		return nil, err
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS health_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL UNIQUE,
			sleep_score INTEGER,
			waist_cm REAL,
			rhr INTEGER,
			nutrition_score REAL
		);`,
		`CREATE TABLE IF NOT EXISTS fitness_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL UNIQUE,
			vo2_max REAL,
			weekly_workouts INTEGER,
			daily_steps INTEGER,
			weekly_mobility INTEGER,
			cardio_recovery INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS cognition_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL UNIQUE,
			dual_n_back_level INTEGER,
			reaction_time INTEGER,
			weekly_mindfulness INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS user_profile (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			birth_date TEXT NOT NULL,
			sex TEXT NOT NULL,
			height_cm REAL NOT NULL
		);`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}
