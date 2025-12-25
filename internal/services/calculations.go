package services

import (
	"database/sql"
	"health-balance/internal/database"
	"health-balance/internal/models"
	"time"
)

// GetAllWeeklyScores calculates scores for all weeks with COMPLETE data
// Only calculates when all three pillars have data for the same date
// This prevents aging tax from being applied multiple times per week
func GetAllWeeklyScores(db *sql.DB) ([]models.MasterScore, error) {
	currentWeekDate := database.GetPreviousSundayDate()
	// Get dates where we have ALL three pillars (INNER JOIN ensures complete data)
	rows, err := db.Query(`
		SELECT h.date 
		FROM health_metrics h
		INNER JOIN fitness_metrics f ON h.date = f.date
		INNER JOIN cognition_metrics c ON h.date = c.date
		WHERE h.date != ?
		ORDER BY h.date DESC
	`, currentWeekDate)
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
	profile, err := database.GetUserProfile(db)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, nil // No profile yet, cannot calculate scores
	}

	// Calculate score for each week (chronological order, oldest first)
	var scores []models.MasterScore
	currentScore := 1000.0 // Starting baseline score

	for i := len(dates) - 1; i >= 0; i-- {
		date := dates[i]

		health, _ := database.GetHealthMetricsByDate(db, date)
		fitness, _ := database.GetFitnessMetricsByDate(db, date)
		cognition, _ := database.GetCognitionMetricsByDate(db, date)

		if health != nil && fitness != nil && cognition != nil {
			rhrBaseline, _ := database.CalculateRHRBaseline(db)
			if rhrBaseline == 0 {
				rhrBaseline = health.RHR
			}

			age := GetAge(profile)
			vo2MaxBaseline := models.GetVO2MaxBaseline(age, profile.Sex)
			reactionBaseline := models.GetReactionTimeBaseline(age)
			whtr := health.WaistCm / profile.HeightCm

			healthComplete := models.HealthMetrics{
				Date:           health.Date,
				SleepScore:     health.SleepScore,
				WaistCm:        health.WaistCm,
				RHR:            health.RHR,
				NutritionScore: health.NutritionScore,
			}

			fitnessComplete := models.FitnessMetrics{
				Date:           fitness.Date,
				VO2Max:         fitness.VO2Max,
				WeeklyWorkouts: fitness.WeeklyWorkouts,
				DailySteps:     fitness.DailySteps,
				WeeklyMobility: fitness.WeeklyMobility,
				CardioRecovery: fitness.CardioRecovery,
			}

			cognitionComplete := models.CognitionMetrics{
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

			scores = append([]models.MasterScore{{
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

func GetCurrentMasterScore(db *sql.DB) (*models.MasterScore, error) {
	scores, err := GetAllWeeklyScores(db)
	if err != nil || len(scores) == 0 {
		return &models.MasterScore{
			Date:  time.Now().Format("2006-01-02"),
			Score: 1000.0, // Starting baseline score
		}, nil
	}
	return &scores[0], nil
}

func CalculateHealthPillar(m models.HealthMetrics, rhrBaseline int, whtr float64) float64 {
	sleepPoints := (m.SleepScore - 75) * 2
	whtrPoints := (0.48 - whtr) * 1000
	rhrPoints := (rhrBaseline - m.RHR) * 5
	nutritionPoints := (m.NutritionScore - 7) * 5

	return float64(sleepPoints) + whtrPoints + float64(rhrPoints) + nutritionPoints
}

func CalculateFitnessPillar(m models.FitnessMetrics, vo2MaxBaseline float64) float64 {
	vo2Points := (m.VO2Max - vo2MaxBaseline) * 20
	workoutPoints := float64(m.WeeklyWorkouts-3) * 20
	stepPoints := float64(m.DailySteps-8000) / 150
	mobilityPoints := float64(m.WeeklyMobility-3) * 10
	recoveryPoints := float64(m.CardioRecovery-20) * 3

	return vo2Points + workoutPoints + stepPoints + mobilityPoints + recoveryPoints
}

func CalculateCognitionPillar(m models.CognitionMetrics, reactionBaseline int) float64 {
	memoryPoints := float64(m.DualNBackLevel-2) * 20
	reactionPoints := float64(reactionBaseline-m.ReactionTime) / 2
	mindfulnessPoints := float64(m.WeeklyMindfulness-3) * 5

	return memoryPoints + reactionPoints + mindfulnessPoints
}

func CalculateMasterScore(
	currentScore float64,
	profile models.UserProfile,
	health models.HealthMetrics,
	fitness models.FitnessMetrics,
	cognition models.CognitionMetrics,
	rhrBaseline int,
	vo2MaxBaseline float64,
	reactionBaseline int,
	whtr float64,
) (newScore float64, healthScore float64, fitnessScore float64, cognitionScore float64, agingTax float64) {

	age := GetAge(&profile)
	weeklyDecayRate := (float64(age*age) / 200.0) / 52.0

	agingTax = currentScore * weeklyDecayRate

	healthScore = CalculateHealthPillar(health, rhrBaseline, whtr)
	fitnessScore = CalculateFitnessPillar(fitness, vo2MaxBaseline)
	cognitionScore = CalculateCognitionPillar(cognition, reactionBaseline)

	weeklyPerformance := healthScore + fitnessScore + cognitionScore

	newScore = (currentScore - agingTax) + weeklyPerformance

	return
}

// GetAge calculates current age from birth date
func GetAge(p *models.UserProfile) int {
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
