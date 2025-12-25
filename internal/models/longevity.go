package models

import (
	"database/sql"
	"time"
)

// MasterScore represents the calculated overall longevity score
type MasterScore struct {
	Date           string
	Score          float64
	HealthScore    float64
	FitnessScore   float64
	CognitionScore float64
	AgingTax       float64
}

// HealthMetrics represents the Health Pillar
type HealthMetrics struct {
	Date           string
	SleepScore     int     // Weekly avg score (0-100)
	WaistCm        float64 // Waist circumference in cm
	RHR            int     // Resting Heart Rate
	NutritionScore float64 // Manual 1-10 score
}

// FitnessMetrics represents the Fitness Pillar
type FitnessMetrics struct {
	Date           string
	VO2Max         float64 // Current VO2 Max
	WeeklyWorkouts int
	DailySteps     int
	WeeklyMobility int
	CardioRecovery int // 60s BPM drop
}

// CognitionMetrics represents the Cognition Pillar
type CognitionMetrics struct {
	Date              string
	DualNBackLevel    int
	ReactionTime      int // Current in ms
	WeeklyMindfulness int
}

// UserProfile stores user-specific data for calculations
type UserProfile struct {
	BirthDate string  // YYYY-MM-DD format
	Sex       string  // "male" or "female"
	HeightCm  float64 // Height in centimeters
}

// GetAge calculates current age from birth date
func (p *UserProfile) GetAge() int {
	birthDate, err := time.Parse("2006-01-02", p.BirthDate)
	if err != nil {
		return 30 // default fallback
	}

	now := time.Now()
	age := now.Year() - birthDate.Year()

	// Adjust if birthday hasn't occurred this year yet
	if now.Month() < birthDate.Month() ||
		(now.Month() == birthDate.Month() && now.Day() < birthDate.Day()) {
		age--
	}

	return age
}

// GetAllWeeklyScores calculates scores for all weeks with data
// GetAllWeeklyScores calculates scores for all weeks with COMPLETE data
// Only calculates when all three pillars have data for the same date
// This prevents aging tax from being applied multiple times per week
func GetAllWeeklyScores(db *sql.DB) ([]MasterScore, error) {
	// Get dates where we have ALL three pillars (INNER JOIN ensures complete data)
	rows, err := db.Query(`
		SELECT h.date 
		FROM health_metrics h
		INNER JOIN fitness_metrics f ON h.date = f.date
		INNER JOIN cognition_metrics c ON h.date = c.date
		ORDER BY h.date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			return nil, err
		}
		dates = append(dates, date)
	}

	// Get profile
	profile, err := GetUserProfile(db)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, nil // No profile yet, cannot calculate scores
	}

	// Calculate score for each week (chronological order, oldest first)
	var scores []MasterScore
	currentScore := 1000.0 // Starting baseline score

	for i := len(dates) - 1; i >= 0; i-- {
		date := dates[i]

		health, _ := GetHealthMetricsByDate(db, date)
		fitness, _ := GetFitnessMetricsByDate(db, date)
		cognition, _ := GetCognitionMetricsByDate(db, date)

		if health != nil && fitness != nil && cognition != nil {
			rhrBaseline, _ := CalculateRHRBaseline(db)
			if rhrBaseline == 0 {
				rhrBaseline = health.RHR
			}

			age := profile.GetAge()
			vo2MaxBaseline := GetVO2MaxBaseline(age, profile.Sex)
			reactionBaseline := GetReactionTimeBaseline(age)
			whtr := health.WaistCm / profile.HeightCm

			healthComplete := HealthMetrics{
				Date:           health.Date,
				SleepScore:     health.SleepScore,
				WaistCm:        health.WaistCm,
				RHR:            health.RHR,
				NutritionScore: health.NutritionScore,
			}

			fitnessComplete := FitnessMetrics{
				Date:           fitness.Date,
				VO2Max:         fitness.VO2Max,
				WeeklyWorkouts: fitness.WeeklyWorkouts,
				DailySteps:     fitness.DailySteps,
				WeeklyMobility: fitness.WeeklyMobility,
				CardioRecovery: fitness.CardioRecovery,
			}

			cognitionComplete := CognitionMetrics{
				Date:              cognition.Date,
				DualNBackLevel:    cognition.DualNBackLevel,
				ReactionTime:      cognition.ReactionTime,
				WeeklyMindfulness: cognition.WeeklyMindfulness,
			}

			newScore, healthScore, fitnessScore, cognitionScore, agingTax := CalculateMasterScore(
				currentScore,
				*profile,
				healthComplete,
				fitnessComplete,
				cognitionComplete,
				rhrBaseline,
				vo2MaxBaseline,
				reactionBaseline,
				whtr,
			)

			scores = append([]MasterScore{{
				Date:           date,
				Score:          newScore,
				HealthScore:    healthScore,
				FitnessScore:   fitnessScore,
				CognitionScore: cognitionScore,
				AgingTax:       agingTax,
			}}, scores...)

			currentScore = newScore
		}
	}

	return scores, nil
}

func GetCurrentMasterScore(db *sql.DB) (*MasterScore, error) {
	scores, err := GetAllWeeklyScores(db)
	if err != nil || len(scores) == 0 {
		return &MasterScore{
			Date:  time.Now().Format("2006-01-02"),
			Score: 1000.0, // Starting baseline score
		}, nil
	}
	return &scores[0], nil
}

func CalculateHealthPillar(m HealthMetrics, rhrBaseline int, whtr float64) float64 {
	sleepPoints := (m.SleepScore - 75) * 2
	whtrPoints := (0.48 - whtr) * 1000
	rhrPoints := (rhrBaseline - m.RHR) * 5
	nutritionPoints := (m.NutritionScore - 7) * 5

	return float64(sleepPoints) + whtrPoints + float64(rhrPoints) + nutritionPoints
}

func CalculateFitnessPillar(m FitnessMetrics, vo2MaxBaseline float64) float64 {
	vo2Points := (m.VO2Max - vo2MaxBaseline) * 20
	workoutPoints := float64(m.WeeklyWorkouts-3) * 20
	stepPoints := float64(m.DailySteps-8000) / 150
	mobilityPoints := float64(m.WeeklyMobility-3) * 10
	recoveryPoints := float64(m.CardioRecovery-20) * 3

	return vo2Points + workoutPoints + stepPoints + mobilityPoints + recoveryPoints
}

func CalculateCognitionPillar(m CognitionMetrics, reactionBaseline int) float64 {
	memoryPoints := float64(m.DualNBackLevel-2) * 20
	reactionPoints := float64(reactionBaseline-m.ReactionTime) / 2
	mindfulnessPoints := float64(m.WeeklyMindfulness-3) * 5

	return memoryPoints + reactionPoints + mindfulnessPoints
}

func CalculateMasterScore(
	currentScore float64,
	profile UserProfile,
	health HealthMetrics,
	fitness FitnessMetrics,
	cognition CognitionMetrics,
	rhrBaseline int,
	vo2MaxBaseline float64,
	reactionBaseline int,
	whtr float64,
) (newScore float64, healthScore float64, fitnessScore float64, cognitionScore float64, agingTax float64) {

	age := profile.GetAge()
	weeklyDecayRate := (float64(age) / 200.0) / 52.0

	agingTax = currentScore * weeklyDecayRate

	healthScore = CalculateHealthPillar(health, rhrBaseline, whtr)
	fitnessScore = CalculateFitnessPillar(fitness, vo2MaxBaseline)
	cognitionScore = CalculateCognitionPillar(cognition, reactionBaseline)

	weeklyPerformance := healthScore + fitnessScore + cognitionScore

	newScore = (currentScore - agingTax) + weeklyPerformance

	return
}
