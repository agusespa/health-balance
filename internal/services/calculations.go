package services

import (
	"errors"
	"fmt"
	"health-balance/internal/database"
	"health-balance/internal/models"
	"math"
	"time"

	"health-balance/internal/utils"
)

const (
	defaultMasterScore      = 1000.0
	scoreAdjustmentRate     = 0.12
	behaviorConsistencySpan = 4
)

func GetCurrentMasterScore(db database.Querier) (*models.MasterScore, error) {
	scores, err := GetAllWeeklyScores(db)

	if err != nil {
		// If error is due to missing profile, return default score instead of error
		if err.Error() == "profile required for master score calculation" {
			return &models.MasterScore{
				Date:  time.Now().Format("2006-01-02"),
				Score: defaultMasterScore,
			}, nil
		}
		return nil, fmt.Errorf("could not get master score: %w", err)
	}

	if len(scores) == 0 {
		return &models.MasterScore{
			Date:  time.Now().Format("2006-01-02"),
			Score: defaultMasterScore,
		}, nil
	}

	return &scores[len(scores)-1], nil
}

func GetAllWeeklyScores(db database.Querier) ([]models.MasterScore, error) {
	allDates, err := db.GetAllDatesWithData()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dates: %w", err)
	}

	profile, err := db.GetUserProfile()
	if err != nil || profile == nil {
		return nil, errors.New("profile required for master score calculation")
	}

	var scores []models.MasterScore
	currentScore := defaultMasterScore
	var healthHistory []models.HealthMetrics
	var fitnessHistory []models.FitnessMetrics
	var cognitionHistory []models.CognitionMetrics

	for i := len(allDates) - 1; i >= 0; i-- {
		date := allDates[i]

		calculationDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			return nil, fmt.Errorf("calculation aborted: invalid metric date %s: %w", date, err)
		}

		age, err := utils.GetAge(profile, calculationDate)
		if err != nil {
			return nil, fmt.Errorf("calculation aborted: invalid profile for date %s: %w", date, err)
		}

		h, _ := db.GetHealthMetricsByDate(date)
		f, _ := db.GetFitnessMetricsByDate(date)
		c, _ := db.GetCognitionMetricsByDate(date)

		// Only calculate if all three pillars exist for this date
		if h == nil || f == nil || c == nil {
			continue
		}

		healthHistory = append(healthHistory, *h)
		fitnessHistory = append(fitnessHistory, *f)
		cognitionHistory = append(cognitionHistory, *c)

		effectiveHealth := smoothHealthBehaviors(healthHistory, *h)
		effectiveFitness := smoothFitnessBehaviors(fitnessHistory, *f)
		effectiveCognition := smoothCognitionBehaviors(cognitionHistory, *c)

		rhrBaseline, _ := db.GetRHRBaselineForDate(date)
		if rhrBaseline == 0 {
			rhrBaseline = h.RHR
		}

		whtr := h.WaistCm / profile.HeightCm

		newScore, hS, fS, cS, tax := CalculateMasterScore(
			currentScore,
			*profile,
			effectiveHealth, effectiveFitness, effectiveCognition,
			rhrBaseline,
			models.GetVO2MaxBaseline(age, profile.Sex),
			whtr,
			calculationDate,
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
	sleepPoints := cappedContribution(float64(m.SleepScore-75), 0.6, 0.9, 8.0, 12.0)
	whtrPoints := cappedContribution(0.48-whtr, 180.0, 260.0, 10.0, 15.0)
	rhrPoints := cappedContribution(float64(rhrBaseline-m.RHR), 1.0, 1.5, 7.0, 10.0)
	bloodPressurePoints := calculateBloodPressurePoints(m)
	nutritionPoints := cappedContribution(m.NutritionScore-7.0, 1.5, 2.0, 4.5, 6.0)

	return sleepPoints + whtrPoints + rhrPoints + bloodPressurePoints + nutritionPoints
}

func CalculateFitnessPillar(m models.FitnessMetrics, vo2MaxBaseline float64) float64 {
	vo2Points := cappedContribution(m.VO2Max-vo2MaxBaseline, 2.5, 3.5, 16.0, 22.0)
	workoutPoints := cappedContribution(float64(m.Workouts-3), 1.5, 2.5, 6.0, 9.0)
	stepPoints := cappedContribution(float64(m.DailySteps-8000)/2000.0, 1.0, 1.5, 3.0, 5.0)
	mobilityPoints := cappedContribution(float64(m.Mobility-3), 1.0, 1.5, 3.0, 4.5)
	recoveryPoints := cappedContribution(float64(m.CardioRecovery-25)/5.0, 1.0, 1.5, 4.0, 6.0)
	strengthPoints := calculateLowerBodyStrengthPoints(m)

	return vo2Points + workoutPoints + stepPoints + mobilityPoints + recoveryPoints + strengthPoints
}

func CalculateCognitionPillar(m models.CognitionMetrics) float64 {
	mindfulnessPoints := cappedContribution(float64(m.Mindfulness-3), 0.6, 1.0, 2.0, 3.0)
	learningPoints := cappedContribution(float64(m.DeepLearning-90)/45.0, 0.6, 1.0, 2.0, 3.0)
	stressPoints := cappedContribution(float64(3-m.StressScore), 1.2, 1.8, 4.0, 6.0)
	socialPoints := cappedContribution(float64(m.SocialDays-4), 0.8, 1.2, 3.0, 4.0)

	return mindfulnessPoints + learningPoints + stressPoints + socialPoints
}

func CalculateMasterScore(
	currentScore float64,
	profile models.UserProfile,
	health models.HealthMetrics,
	fitness models.FitnessMetrics,
	cognition models.CognitionMetrics,
	rhrBaseline int,
	vo2MaxBaseline float64,
	whtr float64,
	calculationDate time.Time,
) (float64, float64, float64, float64, float64) {
	age, _ := utils.GetAge(&profile, calculationDate)
	weeklyDecayRate := (float64(age*age) / 8000.0) / 52.0

	tax := currentScore * weeklyDecayRate
	hScore := CalculateHealthPillar(health, rhrBaseline, whtr)
	fScore := CalculateFitnessPillar(fitness, vo2MaxBaseline)
	cScore := CalculateCognitionPillar(cognition)

	postTaxScore := currentScore - tax
	targetScore := defaultMasterScore + hScore + fScore + cScore
	adjustment := (targetScore - postTaxScore) * scoreAdjustmentRate
	finalScore := postTaxScore + adjustment

	return math.Max(0, finalScore), hScore, fScore, cScore, tax
}

func cappedContribution(delta, positiveSlope, negativeSlope, positiveCap, negativeCap float64) float64 {
	if delta >= 0 {
		return math.Min(delta*positiveSlope, positiveCap)
	}

	return math.Max(delta*negativeSlope, -negativeCap)
}

func smoothHealthBehaviors(history []models.HealthMetrics, current models.HealthMetrics) models.HealthMetrics {
	smoothed := current
	smoothed.SleepScore = int(math.Round(averageLastN(history, behaviorConsistencySpan, func(m models.HealthMetrics) float64 {
		return float64(m.SleepScore)
	})))
	smoothed.NutritionScore = averageLastN(history, behaviorConsistencySpan, func(m models.HealthMetrics) float64 {
		return m.NutritionScore
	})
	return smoothed
}

func smoothFitnessBehaviors(history []models.FitnessMetrics, current models.FitnessMetrics) models.FitnessMetrics {
	smoothed := current
	smoothed.Workouts = int(math.Round(averageLastN(history, behaviorConsistencySpan, func(m models.FitnessMetrics) float64 {
		return float64(m.Workouts)
	})))
	smoothed.DailySteps = int(math.Round(averageLastN(history, behaviorConsistencySpan, func(m models.FitnessMetrics) float64 {
		return float64(m.DailySteps)
	})))
	smoothed.Mobility = int(math.Round(averageLastN(history, behaviorConsistencySpan, func(m models.FitnessMetrics) float64 {
		return float64(m.Mobility)
	})))
	return smoothed
}

func smoothCognitionBehaviors(history []models.CognitionMetrics, current models.CognitionMetrics) models.CognitionMetrics {
	smoothed := current
	smoothed.Mindfulness = int(math.Round(averageLastN(history, behaviorConsistencySpan, func(m models.CognitionMetrics) float64 {
		return float64(m.Mindfulness)
	})))
	smoothed.DeepLearning = int(math.Round(averageLastN(history, behaviorConsistencySpan, func(m models.CognitionMetrics) float64 {
		return float64(m.DeepLearning)
	})))
	smoothed.StressScore = int(math.Round(averageLastN(history, behaviorConsistencySpan, func(m models.CognitionMetrics) float64 {
		return float64(m.StressScore)
	})))
	smoothed.SocialDays = int(math.Round(averageLastN(history, behaviorConsistencySpan, func(m models.CognitionMetrics) float64 {
		return float64(m.SocialDays)
	})))
	return smoothed
}

func averageLastN[T any](items []T, window int, value func(T) float64) float64 {
	if len(items) == 0 {
		return 0
	}

	start := len(items) - window
	if start < 0 {
		start = 0
	}

	var total float64
	for _, item := range items[start:] {
		total += value(item)
	}

	return total / float64(len(items[start:]))
}

func calculateBloodPressurePoints(m models.HealthMetrics) float64 {
	if m.SystolicBP <= 0 || m.DiastolicBP <= 0 {
		return 0
	}

	systolicPoints := cappedContribution(float64(120-m.SystolicBP)/5.0, 1.0, 1.5, 5.0, 8.0)
	diastolicPoints := cappedContribution(float64(80-m.DiastolicBP)/3.0, 1.0, 1.5, 4.0, 6.0)
	return systolicPoints + diastolicPoints
}

func calculateLowerBodyStrengthPoints(m models.FitnessMetrics) float64 {
	if m.LowerBodyWeight <= 0 || m.LowerBodyReps <= 0 {
		return 0
	}

	legPressLoad := m.LowerBodyWeight * float64(m.LowerBodyReps)
	return cappedContribution((legPressLoad-1200.0)/200.0, 1.2, 1.5, 6.0, 8.0)
}
