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
			body_weight_kg REAL,
			rhr INTEGER,
			systolic_bp INTEGER,
			diastolic_bp INTEGER,
			nutrition_score REAL
		);`,
		`CREATE TABLE IF NOT EXISTS fitness_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL UNIQUE,
			vo2_max REAL,
			workouts INTEGER,
			daily_steps INTEGER,
			mobility INTEGER,
			cardio_recovery INTEGER,
			lower_body_weight REAL,
			lower_body_reps INTEGER,
			dead_hang_seconds INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS cognition_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL UNIQUE,
			mindfulness INTEGER,
			deep_learning INTEGER,
			stress_score INTEGER,
			social_days INTEGER
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

	return nil
}
