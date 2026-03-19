package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"time"

	"health-balance/internal/database"
	"health-balance/internal/models"
	"health-balance/internal/utils"
)

const defaultDBPath = "./data/health.db"

type weekSample struct {
	health    models.HealthMetrics
	fitness   models.FitnessMetrics
	cognition models.CognitionMetrics
}

func main() {
	var (
		dbPath             string
		weeks              int
		reset              bool
		includeCurrentWeek bool
	)

	flag.StringVar(&dbPath, "db", defaultDBPath, "SQLite database path")
	flag.IntVar(&weeks, "weeks", 14, "number of recent weeks to seed")
	flag.BoolVar(&reset, "reset", false, "clear existing metrics and profile before seeding")
	flag.BoolVar(&includeCurrentWeek, "include-current-week", false, "seed the current week instead of leaving it empty")
	flag.Parse()

	if weeks < 4 {
		log.Fatal("weeks must be at least 4")
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatalf("failed to create data directory: %v", err)
	}

	db, err := database.Init(dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}()

	if reset {
		if err := resetSeedTables(db); err != nil {
			log.Fatalf("failed to reset database: %v", err)
		}
	}

	if err := db.SaveUserProfile(models.UserProfile{
		BirthDate: "1990-06-15",
		Sex:       "male",
		HeightCm:  180,
	}); err != nil {
		log.Fatalf("failed to seed profile: %v", err)
	}

	currentWeek, err := time.Parse("2006-01-02", utils.GetCurrentWeekSundayDate())
	if err != nil {
		log.Fatalf("failed to parse current week: %v", err)
	}

	skippedOffsets := map[int]bool{
		4: true,
		9: true,
	}

	seededWeeks := 0
	for offset := weeks - 1; offset >= 0; offset-- {
		if offset == 0 && !includeCurrentWeek {
			continue
		}
		if skippedOffsets[offset] {
			continue
		}

		date := currentWeek.AddDate(0, 0, -7*offset).Format("2006-01-02")
		recency := normalizedRecency(offset, weeks)
		sample := buildWeekSample(recency, offset)

		if err := upsertWeek(db, date, sample); err != nil {
			log.Fatalf("failed to seed week %s: %v", date, err)
		}
		seededWeeks++
	}

	fmt.Printf("Seeded %d weeks into %s\n", seededWeeks, dbPath)
	if reset {
		fmt.Println("Existing profile and metric rows were cleared before seeding.")
	}
	if !includeCurrentWeek {
		fmt.Println("The current week was left empty on purpose so you can test the copy/rollover flow.")
	}
	fmt.Println("Intentional gaps were added a few weeks back to exercise missed-week scoring.")
}

func resetSeedTables(db *database.DB) error {
	statements := []string{
		"DELETE FROM health_metrics",
		"DELETE FROM fitness_metrics",
		"DELETE FROM cognition_metrics",
		"DELETE FROM user_profile",
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

func upsertWeek(db *database.DB, date string, sample weekSample) error {
	if _, err := db.Exec(`
		INSERT INTO health_metrics (date, sleep_score, waist_cm, rhr, systolic_bp, diastolic_bp, nutrition_score)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			sleep_score = excluded.sleep_score,
			waist_cm = excluded.waist_cm,
			rhr = excluded.rhr,
			systolic_bp = excluded.systolic_bp,
			diastolic_bp = excluded.diastolic_bp,
			nutrition_score = excluded.nutrition_score
	`, date, sample.health.SleepScore, sample.health.WaistCm, sample.health.RHR, sample.health.SystolicBP, sample.health.DiastolicBP, sample.health.NutritionScore); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO fitness_metrics (date, vo2_max, workouts, daily_steps, mobility, cardio_recovery, lower_body_weight, lower_body_reps)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			vo2_max = excluded.vo2_max,
			workouts = excluded.workouts,
			daily_steps = excluded.daily_steps,
			mobility = excluded.mobility,
			cardio_recovery = excluded.cardio_recovery,
			lower_body_weight = excluded.lower_body_weight,
			lower_body_reps = excluded.lower_body_reps
	`, date, sample.fitness.VO2Max, sample.fitness.Workouts, sample.fitness.DailySteps, sample.fitness.Mobility, sample.fitness.CardioRecovery, sample.fitness.LowerBodyWeight, sample.fitness.LowerBodyReps); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO cognition_metrics (date, mindfulness, deep_learning, stress_score, social_days)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			mindfulness = excluded.mindfulness,
			deep_learning = excluded.deep_learning,
			stress_score = excluded.stress_score,
			social_days = excluded.social_days
	`, date, sample.cognition.Mindfulness, sample.cognition.DeepLearning, sample.cognition.StressScore, sample.cognition.SocialDays); err != nil {
		return err
	}

	return nil
}

func buildWeekSample(recency float64, offset int) weekSample {
	waveA := math.Sin(float64(offset) * 0.9)
	waveB := math.Cos(float64(offset) * 0.6)

	return weekSample{
		health: models.HealthMetrics{
			SleepScore:     clampInt(72+int(math.Round(recency*10))+int(math.Round(waveA*3)), 66, 90),
			WaistCm:        clampFloat(87.2-recency*2.4+(waveB*0.4), 82.5, 90.0),
			RHR:            clampInt(66-int(math.Round(recency*6))+int(math.Round(waveA)), 56, 69),
			SystolicBP:     clampInt(124-int(math.Round(recency*5))+int(math.Round(waveB*2)), 114, 130),
			DiastolicBP:    clampInt(82-int(math.Round(recency*4))+int(math.Round(waveA)), 72, 88),
			NutritionScore: clampFloat(6.6+recency*1.6+(waveB*0.25), 5.8, 8.8),
		},
		fitness: models.FitnessMetrics{
			VO2Max:          clampFloat(38.5+recency*6.0+(waveA*0.8), 36.5, 47.0),
			Workouts:        clampInt(2+int(math.Round(recency*2))+positiveSwing(offset, 3), 1, 5),
			DailySteps:      clampInt(6900+int(math.Round(recency*2600))+int(math.Round(waveB*700)), 5500, 12000),
			Mobility:        clampInt(1+int(math.Round(recency*2))+positiveSwing(offset+1, 4), 1, 4),
			CardioRecovery:  clampInt(21+int(math.Round(recency*6))+int(math.Round(waveA*2)), 17, 32),
			LowerBodyWeight: clampFloat(165+recency*38+(waveB*6), 150, 230),
			LowerBodyReps:   clampInt(8+positiveSwing(offset+2, 5)+int(math.Round(recency*2)), 8, 13),
		},
		cognition: models.CognitionMetrics{
			Mindfulness:  clampInt(1+int(math.Round(recency*2))+positiveSwing(offset+3, 4), 1, 5),
			DeepLearning: clampInt(50+int(math.Round(recency*55))+int(math.Round(waveA*12)), 30, 130),
			StressScore:  clampInt(4-int(math.Round(recency*2))+positiveSwing(offset+4, 6)-1, 1, 4),
			SocialDays:   clampInt(2+int(math.Round(recency*2))+positiveSwing(offset+5, 3), 1, 6),
		},
	}
}

func normalizedRecency(offset, weeks int) float64 {
	if weeks <= 1 {
		return 1
	}
	return float64((weeks-1)-offset) / float64(weeks-1)
}

func positiveSwing(offset, modulo int) int {
	if modulo <= 0 {
		return 0
	}
	return offset % modulo / (modulo/2 + 1)
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func clampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return math.Round(value*10) / 10
}
