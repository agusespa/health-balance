package models

import (
	"database/sql"
	"time"
)

func GetPreviousSundayDate() string {
	now := time.Now()
	weekday := now.Weekday()
	daysSinceSunday := int(weekday)
	previousSunday := now.AddDate(0, 0, -daysSinceSunday)
	return previousSunday.Format("2006-01-02")
}

func SaveHealthMetrics(db *sql.DB, m HealthMetrics) error {
	date := GetPreviousSundayDate()
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

func SaveFitnessMetrics(db *sql.DB, m FitnessMetrics) error {
	date := GetPreviousSundayDate()
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

func SaveCognitionMetrics(db *sql.DB, m CognitionMetrics) error {
	date := GetPreviousSundayDate()
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

func GetRecentHealthMetrics(db *sql.DB, limit int) ([]HealthMetrics, error) {
	currentWeekDate := GetPreviousSundayDate()
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
	defer rows.Close()

	var metrics []HealthMetrics
	for rows.Next() {
		var m HealthMetrics
		err := rows.Scan(&m.Date, &m.SleepScore, &m.WaistCm, &m.RHR, &m.NutritionScore)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

func GetRecentFitnessMetrics(db *sql.DB, limit int) ([]FitnessMetrics, error) {
	currentWeekDate := GetPreviousSundayDate()
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
	defer rows.Close()

	var metrics []FitnessMetrics
	for rows.Next() {
		var m FitnessMetrics
		err := rows.Scan(&m.Date, &m.VO2Max, &m.WeeklyWorkouts, &m.DailySteps, &m.WeeklyMobility, &m.CardioRecovery)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

func GetRecentCognitionMetrics(db *sql.DB, limit int) ([]CognitionMetrics, error) {
	currentWeekDate := GetPreviousSundayDate()
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
	defer rows.Close()

	var metrics []CognitionMetrics
	for rows.Next() {
		var m CognitionMetrics
		err := rows.Scan(&m.Date, &m.DualNBackLevel, &m.ReactionTime, &m.WeeklyMindfulness)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

func GetHealthMetricsByDate(db *sql.DB, date string) (*HealthMetrics, error) {
	var m HealthMetrics
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

func GetFitnessMetricsByDate(db *sql.DB, date string) (*FitnessMetrics, error) {
	var m FitnessMetrics
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

func GetCognitionMetricsByDate(db *sql.DB, date string) (*CognitionMetrics, error) {
	var m CognitionMetrics
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

// CalculateRHRBaseline calculates the 3-month average RHR
func CalculateRHRBaseline(db *sql.DB) (int, error) {
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

func GetUserProfile(db *sql.DB) (*UserProfile, error) {
	var profile UserProfile
	err := db.QueryRow("SELECT id, birth_date, sex, height_cm FROM user_profile LIMIT 1").
		Scan(&profile.Id, &profile.BirthDate, &profile.Sex, &profile.HeightCm)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &profile, err
}

func SaveUserProfile(db *sql.DB, profile UserProfile) error {
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
