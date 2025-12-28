package main

import (
	"fmt"
	"health-balance/internal/database"
	"health-balance/internal/models"
	"log"
	"math/rand"
	"time"
)

func main() {
	db, err := database.Init("../data/health.db")
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()

	// Create User Profile
	userProfile := models.UserProfile{
		BirthDate: "1990-01-01",
		Sex:       "Male",
		HeightCm:  180,
	}
	if err := db.SaveUserProfile(userProfile); err != nil {
		log.Fatalf("failed to save user profile: %v", err)
	}
	fmt.Println("User profile created.")

	// Generate data for the last 12 weeks
	for i := 12; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i*7)
		sunday := getPreviousSunday(date)
		dateStr := sunday.Format("2006-01-02")

		healthMetrics := models.HealthMetrics{
			Date:           dateStr,
			SleepScore:     rand.Intn(31) + 65, // 65-95
			WaistCm:        80 + rand.Float64()*5,
			RHR:            55 + rand.Intn(10), // 55-64
			NutritionScore: 6.5 + rand.Float64()*2,
		}

		fitnessMetrics := models.FitnessMetrics{
			Date:           dateStr,
			VO2Max:         40 + rand.Float64()*5,
			WeeklyWorkouts: rand.Intn(3) + 2, // 2-4
			DailySteps:     8000 + rand.Intn(4000),
			WeeklyMobility: rand.Intn(3) + 1, // 1-3
			CardioRecovery: 20 + rand.Intn(10),
		}

		cognitionMetrics := models.CognitionMetrics{
			Date:              dateStr,
			DualNBackLevel:    rand.Intn(3) + 2, // 2-4
			ReactionTime:      220 + rand.Intn(40),
			WeeklyMindfulness: rand.Intn(4) + 2, // 2-5
		}

		// Custom save functions to insert with a specific date
		if err := saveHealthMetricsWithDate(db, healthMetrics); err != nil {
			log.Printf("failed to save health metrics for %s: %v", dateStr, err)
		}
		if err := saveFitnessMetricsWithDate(db, fitnessMetrics); err != nil {
			log.Printf("failed to save fitness metrics for %s: %v", dateStr, err)
		}
		if err := saveCognitionMetricsWithDate(db, cognitionMetrics); err != nil {
			log.Printf("failed to save cognition metrics for %s: %v", dateStr, err)
		}
	}

	fmt.Println("Database populated with 12 weeks of sample data.")
}

func getPreviousSunday(t time.Time) time.Time {
	weekday := t.Weekday()
	if weekday == time.Sunday {
		return t
	}
	return t.AddDate(0, 0, -int(weekday))
}

func saveHealthMetricsWithDate(db *database.DB, m models.HealthMetrics) error {
	_, err := db.Exec(`
		INSERT INTO health_metrics (date, sleep_score, waist_cm, rhr, nutrition_score)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			sleep_score = excluded.sleep_score,
			waist_cm = excluded.waist_cm,
			rhr = excluded.rhr,
			nutrition_score = excluded.nutrition_score
	`, m.Date, m.SleepScore, m.WaistCm, m.RHR, m.NutritionScore)
	return err
}

func saveFitnessMetricsWithDate(db *database.DB, m models.FitnessMetrics) error {
	_, err := db.Exec(`
		INSERT INTO fitness_metrics (date, vo2_max, weekly_workouts, daily_steps, weekly_mobility, cardio_recovery)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			vo2_max = excluded.vo2_max,
			weekly_workouts = excluded.weekly_workouts,
			daily_steps = excluded.daily_steps,
			weekly_mobility = excluded.weekly_mobility,
			cardio_recovery = excluded.cardio_recovery
	`, m.Date, m.VO2Max, m.WeeklyWorkouts, m.DailySteps, m.WeeklyMobility, m.CardioRecovery)
	return err
}

func saveCognitionMetricsWithDate(db *database.DB, m models.CognitionMetrics) error {
	_, err := db.Exec(`
		INSERT INTO cognition_metrics (date, dual_n_back_level, reaction_time, weekly_mindfulness)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			dual_n_back_level = excluded.dual_n_back_level,
			reaction_time = excluded.reaction_time,
			weekly_mindfulness = excluded.weekly_mindfulness
	`, m.Date, m.DualNBackLevel, m.ReactionTime, m.WeeklyMindfulness)
	return err
}
