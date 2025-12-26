package services

import (
	"database/sql"
	"errors"
	"fmt"
	"health-balance/internal/database"
	"health-balance/internal/models"
	"health-balance/internal/utils"
	"math"
	"time"
)

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

func GetAllWeeklyScores(db *sql.DB) ([]models.MasterScore, error) {
	allDates, err := database.GetAllDatesWithData(db)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dates: %w", err)
	}

	profile, err := database.GetUserProfile(db)
	if err != nil || profile == nil {
		return nil, errors.New("profile required for master score calculation")
	}

	var scores []models.MasterScore
	currentScore := 1000.0

	for i := len(allDates) - 1; i >= 0; i-- {
		date := allDates[i]

		age, err := utils.GetAge(profile)
		if err != nil {
			return nil, fmt.Errorf("calculation aborted: invalid profile for date %s: %w", date, err)
		}

		h, _ := database.GetHealthMetricsByDate(db, date)
		f, _ := database.GetFitnessMetricsByDate(db, date)
		c, _ := database.GetCognitionMetricsByDate(db, date)

		// Only calculate if all three pillars exist for this date
		if h == nil || f == nil || c == nil {
			continue
		}

		rhrBaseline, _ := database.GetRHRBaseline(db)
		if rhrBaseline == 0 {
			rhrBaseline = h.RHR
		}

		whtr := h.WaistCm / profile.HeightCm

		newScore, hS, fS, cS, tax := CalculateMasterScore(
			currentScore,
			*profile,
			*h, *f, *c,
			rhrBaseline,
			models.GetVO2MaxBaseline(age, profile.Sex),
			models.GetReactionTimeBaseline(age),
			whtr,
		)

		scores = append(scores, models.MasterScore{
			Date:           date,
			Score:          newScore,
			HealthScore:    hS,
			FitnessScore:   fS,
			CognitionScore: cS,
			AgingTax:       tax,
		})

		currentScore = newScore
	}

	return scores, nil
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
) (float64, float64, float64, float64, float64) {
	age, _ := utils.GetAge(&profile)
	weeklyDecayRate := (float64(age*age) / 8000.0) / 52.0

	tax := currentScore * weeklyDecayRate
	hScore := CalculateHealthPillar(health, rhrBaseline, whtr)
	fScore := CalculateFitnessPillar(fitness, vo2MaxBaseline)
	cScore := CalculateCognitionPillar(cognition, reactionBaseline)

	performance := hScore + fScore + cScore
	finalScore := (currentScore - tax) + performance

	return math.Max(0, finalScore), hScore, fScore, cScore, tax
}
