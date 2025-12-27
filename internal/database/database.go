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
		`CREATE TABLE IF NOT EXISTS push_subscriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			endpoint TEXT NOT NULL UNIQUE,
			p256dh TEXT NOT NULL,
			auth TEXT NOT NULL,
			reminder_day INTEGER NOT NULL DEFAULT 0,
			reminder_time TEXT NOT NULL DEFAULT '15:00',
			timezone TEXT NOT NULL DEFAULT 'UTC'
		);`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	// Migration: Add timezone column if it doesn't exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('push_subscriptions') WHERE name='timezone'").Scan(&count)
	if err == nil && count == 0 {
		if _, err := db.Exec("ALTER TABLE push_subscriptions ADD COLUMN timezone TEXT NOT NULL DEFAULT 'UTC'"); err != nil {
			return err
		}
	}

	return nil
}
