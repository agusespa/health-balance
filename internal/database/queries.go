package database

import (
	"database/sql"
	"health-balance/internal/models"
)

type Querier interface {
	GetUserProfile() (*models.UserProfile, error)
	SaveUserProfile(profile models.UserProfile) error
	SaveIngest(run models.IngestRun, values []models.DailyMetric, clear []models.DayMetric) (int64, error)
	GetDailyMetrics(from, to string, metrics ...string) ([]models.DailyMetric, error)
	GetLatestDataDate() (string, error)
	SaveInsight(in models.Insight) error
	GetLatestInsight() (*models.Insight, error)
	Close() error
}

func (db *DB) GetUserProfile() (*models.UserProfile, error) {
	var profile models.UserProfile
	err := db.QueryRow("SELECT id, birth_date, sex, height_cm FROM user_profile ORDER BY id DESC LIMIT 1").
		Scan(&profile.Id, &profile.BirthDate, &profile.Sex, &profile.HeightCm)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &profile, err
}

func (db *DB) SaveUserProfile(profile models.UserProfile) error {
	var existingID int
	err := db.QueryRow("SELECT id FROM user_profile LIMIT 1").Scan(&existingID)

	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if existingID > 0 {
		// Update existing profile
		_, err = db.Exec(`
			UPDATE user_profile SET birth_date = ?, sex = ?, height_cm = ? WHERE id = ?
		`, profile.BirthDate, profile.Sex, profile.HeightCm, existingID)
	} else {
		// Insert new profile
		_, err = db.Exec(`
			INSERT INTO user_profile (birth_date, sex, height_cm) VALUES (?, ?, ?)
		`, profile.BirthDate, profile.Sex, profile.HeightCm)
	}

	return err
}
