package database

import (
	"database/sql"
	"health-balance/internal/models"
	"health-balance/internal/utils"
	"log"
	"time"
)

type Querier interface {
	GetAllDatesWithData() ([]string, error)
	GetRecentHealthMetrics(limit int) ([]models.HealthMetrics, error)
	GetRecentFitnessMetrics(limit int) ([]models.FitnessMetrics, error)
	GetRecentCognitionMetrics(limit int) ([]models.CognitionMetrics, error)
	SaveHealthMetrics(m models.HealthMetrics) error
	SaveFitnessMetrics(m models.FitnessMetrics) error
	SaveCognitionMetrics(m models.CognitionMetrics) error
	GetHealthMetricsByDate(date string) (*models.HealthMetrics, error)
	GetFitnessMetricsByDate(date string) (*models.FitnessMetrics, error)
	GetCognitionMetricsByDate(date string) (*models.CognitionMetrics, error)
	GetRHRBaseline() (int, error)
	GetUserProfile() (*models.UserProfile, error)
	SaveUserProfile(profile models.UserProfile) error
	Close() error
}

// GetAllDatesWithData retrieves a unique, sorted list of all dates that have an entry in any of the three metric tables
func (db *DB) GetAllDatesWithData() ([]string, error) {
	query := `
		SELECT date FROM health_metrics
		UNION
		SELECT date FROM fitness_metrics
		UNION
		SELECT date FROM cognition_metrics
		ORDER BY date DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()

	var dates []string
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			return nil, err
		}
		dates = append(dates, date)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return dates, nil
}

func (db *DB) GetRecentHealthMetrics(limit int) ([]models.HealthMetrics, error) {
	currentWeekDate := utils.GetPreviousSundayDate()
	rows, err := db.Query(`
		SELECT date, sleep_score, waist_cm, rhr, nutrition_score
		FROM health_metrics
		WHERE date != ?
		ORDER BY date DESC
		LIMIT ?
	`, currentWeekDate, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()

	var metrics []models.HealthMetrics
	for rows.Next() {
		var m models.HealthMetrics
		err := rows.Scan(&m.Date, &m.SleepScore, &m.WaistCm, &m.RHR, &m.NutritionScore)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

func (db *DB) GetRecentFitnessMetrics(limit int) ([]models.FitnessMetrics, error) {
	currentWeekDate := utils.GetPreviousSundayDate()
	rows, err := db.Query(`
		SELECT date, vo2_max, weekly_workouts, daily_steps, weekly_mobility, cardio_recovery
		FROM fitness_metrics
		WHERE date != ?
		ORDER BY date DESC
		LIMIT ?
	`, currentWeekDate, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()

	var metrics []models.FitnessMetrics
	for rows.Next() {
		var m models.FitnessMetrics
		err := rows.Scan(&m.Date, &m.VO2Max, &m.WeeklyWorkouts, &m.DailySteps, &m.WeeklyMobility, &m.CardioRecovery)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

func (db *DB) GetRecentCognitionMetrics(limit int) ([]models.CognitionMetrics, error) {
	currentWeekDate := utils.GetPreviousSundayDate()
	rows, err := db.Query(`
		SELECT date, dual_n_back_level, reaction_time, weekly_mindfulness
		FROM cognition_metrics
		WHERE date != ?
		ORDER BY date DESC
		LIMIT ?
	`, currentWeekDate, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()

	var metrics []models.CognitionMetrics
	for rows.Next() {
		var m models.CognitionMetrics
		err := rows.Scan(&m.Date, &m.DualNBackLevel, &m.ReactionTime, &m.WeeklyMindfulness)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

func (db *DB) SaveHealthMetrics(m models.HealthMetrics) error {
	date := utils.GetPreviousSundayDate()
	_, err := db.Exec(`
		INSERT INTO health_metrics (date, sleep_score, waist_cm, rhr, nutrition_score)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			sleep_score = excluded.sleep_score,
			waist_cm = excluded.waist_cm,
			rhr = excluded.rhr,
			nutrition_score = excluded.nutrition_score
	`, date, m.SleepScore, m.WaistCm, m.RHR, m.NutritionScore)
	return err
}

func (db *DB) SaveFitnessMetrics(m models.FitnessMetrics) error {
	date := utils.GetPreviousSundayDate()
	_, err := db.Exec(`
		INSERT INTO fitness_metrics (date, vo2_max, weekly_workouts, daily_steps, weekly_mobility, cardio_recovery)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			vo2_max = excluded.vo2_max,
			weekly_workouts = excluded.weekly_workouts,
			daily_steps = excluded.daily_steps,
			weekly_mobility = excluded.weekly_mobility,
			cardio_recovery = excluded.cardio_recovery
	`, date, m.VO2Max, m.WeeklyWorkouts, m.DailySteps, m.WeeklyMobility, m.CardioRecovery)
	return err
}

func (db *DB) SaveCognitionMetrics(m models.CognitionMetrics) error {
	date := utils.GetPreviousSundayDate()
	_, err := db.Exec(`
		INSERT INTO cognition_metrics (date, dual_n_back_level, reaction_time, weekly_mindfulness)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			dual_n_back_level = excluded.dual_n_back_level,
			reaction_time = excluded.reaction_time,
			weekly_mindfulness = excluded.weekly_mindfulness
	`, date, m.DualNBackLevel, m.ReactionTime, m.WeeklyMindfulness)
	return err
}

func (db *DB) GetHealthMetricsByDate(date string) (*models.HealthMetrics, error) {
	var m models.HealthMetrics
	err := db.QueryRow(`
		SELECT date, sleep_score, waist_cm, rhr, nutrition_score
		FROM health_metrics
		WHERE date = ?
	`, date).Scan(&m.Date, &m.SleepScore, &m.WaistCm, &m.RHR, &m.NutritionScore)

	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (db *DB) GetFitnessMetricsByDate(date string) (*models.FitnessMetrics, error) {
	var m models.FitnessMetrics
	err := db.QueryRow(`
		SELECT date, vo2_max, weekly_workouts, daily_steps, weekly_mobility, cardio_recovery
		FROM fitness_metrics
		WHERE date = ?
	`, date).Scan(&m.Date, &m.VO2Max, &m.WeeklyWorkouts, &m.DailySteps, &m.WeeklyMobility, &m.CardioRecovery)

	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (db *DB) GetCognitionMetricsByDate(date string) (*models.CognitionMetrics, error) {
	var m models.CognitionMetrics
	err := db.QueryRow(`
		SELECT date, dual_n_back_level, reaction_time, weekly_mindfulness
		FROM cognition_metrics
		WHERE date = ?
	`, date).Scan(&m.Date, &m.DualNBackLevel, &m.ReactionTime, &m.WeeklyMindfulness)

	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetRHRBaseline calculates the 3-month average RHR
func (db *DB) GetRHRBaseline() (int, error) {
	threeMonthsAgo := time.Now().AddDate(0, -3, 0).Format("2006-01-02")

	var baseline int
	err := db.QueryRow(`
		SELECT AVG(rhr)
		FROM health_metrics
		WHERE date >= ?
	`, threeMonthsAgo).Scan(&baseline)

	if err != nil {
		return 0, err
	}

	return baseline, nil
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
	// Check if a profile already exists
	var existingID int
	err := db.QueryRow("SELECT id FROM user_profile LIMIT 1").Scan(&existingID)

	if err != nil && err != sql.ErrNoRows {
		return err // Handle potential database errors
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
