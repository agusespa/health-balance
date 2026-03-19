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
	behaviorDecayRate       = 0.5
	stableDriftRate         = 0.25
	subjectiveDriftRate     = 0.5
	stableCarryWeeks        = 2
	subjectiveCarryWeeks    = 1
	neutralSleepScore       = 75.0
	neutralNutritionScore   = 7.0
	neutralStressScore      = 3.0
	neutralSystolicBP       = 120.0
	neutralDiastolicBP      = 80.0
	neutralCardioRecovery   = 25.0
	neutralLegPressWeight   = 120.0
	neutralLegPressReps     = 10.0
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
	if len(allDates) == 0 {
		return nil, nil
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

	startDate, err := time.Parse("2006-01-02", allDates[len(allDates)-1])
	if err != nil {
		return nil, fmt.Errorf("calculation aborted: invalid metric date %s: %w", allDates[len(allDates)-1], err)
	}

	endDate, err := time.Parse("2006-01-02", utils.GetCurrentWeekSundayDate())
	if err != nil {
		return nil, fmt.Errorf("calculation aborted: invalid current week date: %w", err)
	}
	if endDate.Before(startDate) {
		endDate = startDate
	}

	var (
		lastHealth      models.HealthMetrics
		lastFitness     models.FitnessMetrics
		lastCognition   models.CognitionMetrics
		healthReady     bool
		fitnessReady    bool
		cognitionReady  bool
		healthMissed    int
		fitnessMissed   int
		cognitionMissed int
	)

	for _, calculationDate := range expandWeeklyDates(startDate, endDate) {
		date := calculationDate.Format("2006-01-02")

		age, err := utils.GetAge(profile, calculationDate)
		if err != nil {
			return nil, fmt.Errorf("calculation aborted: invalid profile for date %s: %w", date, err)
		}

		h, _ := db.GetHealthMetricsByDate(date)
		f, _ := db.GetFitnessMetricsByDate(date)
		c, _ := db.GetCognitionMetricsByDate(date)

		rhrBaseline, _ := db.GetRHRBaselineForDate(date)
		if h != nil && rhrBaseline == 0 {
			rhrBaseline = h.RHR
		}

		if h != nil {
			lastHealth = *h
			healthReady = true
			healthMissed = 0
		} else if healthReady {
			healthMissed++
			lastHealth = imputeHealthMetrics(lastHealth, healthMissed, *profile, rhrBaseline)
		}

		if f != nil {
			lastFitness = *f
			fitnessReady = true
			fitnessMissed = 0
		} else if fitnessReady {
			fitnessMissed++
			lastFitness = imputeFitnessMetrics(lastFitness, fitnessMissed, age, *profile)
		}

		if c != nil {
			lastCognition = *c
			cognitionReady = true
			cognitionMissed = 0
		} else if cognitionReady {
			cognitionMissed++
			lastCognition = imputeCognitionMetrics(lastCognition, cognitionMissed)
		}

		if healthReady {
			lastHealth.Date = date
			healthHistory = append(healthHistory, lastHealth)
		}
		if fitnessReady {
			lastFitness.Date = date
			fitnessHistory = append(fitnessHistory, lastFitness)
		}
		if cognitionReady {
			lastCognition.Date = date
			cognitionHistory = append(cognitionHistory, lastCognition)
		}

		if !healthReady || !fitnessReady || !cognitionReady {
			continue
		}

		effectiveHealth := smoothHealthBehaviors(healthHistory, lastHealth)
		effectiveFitness := smoothFitnessBehaviors(fitnessHistory, lastFitness)
		effectiveCognition := smoothCognitionBehaviors(cognitionHistory, lastCognition)

		if rhrBaseline == 0 {
			rhrBaseline = lastHealth.RHR
		}

		whtr := lastHealth.WaistCm / profile.HeightCm

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

func expandWeeklyDates(start, end time.Time) []time.Time {
	var dates []time.Time
	for current := start; !current.After(end); current = current.AddDate(0, 0, 7) {
		dates = append(dates, current)
	}
	return dates
}

func imputeHealthMetrics(previous models.HealthMetrics, missedWeeks int, profile models.UserProfile, rhrBaseline int) models.HealthMetrics {
	imputed := previous
	imputed.SleepScore = imputeSubjectiveInt(previous.SleepScore, neutralSleepScore, missedWeeks)
	imputed.NutritionScore = imputeSubjectiveFloat(previous.NutritionScore, neutralNutritionScore, missedWeeks)
	imputed.WaistCm = imputeStableFloat(previous.WaistCm, profile.HeightCm*0.48, missedWeeks)

	targetRHR := float64(rhrBaseline)
	if targetRHR == 0 {
		targetRHR = float64(previous.RHR)
	}
	imputed.RHR = imputeStableInt(previous.RHR, targetRHR, missedWeeks)
	imputed.SystolicBP = imputeStableInt(previous.SystolicBP, neutralSystolicBP, missedWeeks)
	imputed.DiastolicBP = imputeStableInt(previous.DiastolicBP, neutralDiastolicBP, missedWeeks)
	return imputed
}

func imputeFitnessMetrics(previous models.FitnessMetrics, missedWeeks int, age int, profile models.UserProfile) models.FitnessMetrics {
	imputed := previous
	imputed.Workouts = decayBehaviorInt(previous.Workouts)
	imputed.DailySteps = decayBehaviorInt(previous.DailySteps)
	imputed.Mobility = decayBehaviorInt(previous.Mobility)
	imputed.VO2Max = imputeStableFloat(previous.VO2Max, models.GetVO2MaxBaseline(age, profile.Sex), missedWeeks)
	imputed.CardioRecovery = imputeStableInt(previous.CardioRecovery, neutralCardioRecovery, missedWeeks)
	imputed.LowerBodyWeight = imputeStableFloat(previous.LowerBodyWeight, neutralLegPressWeight, missedWeeks)
	imputed.LowerBodyReps = imputeStableInt(previous.LowerBodyReps, neutralLegPressReps, missedWeeks)
	return imputed
}

func imputeCognitionMetrics(previous models.CognitionMetrics, missedWeeks int) models.CognitionMetrics {
	imputed := previous
	imputed.Mindfulness = decayBehaviorInt(previous.Mindfulness)
	imputed.DeepLearning = decayBehaviorInt(previous.DeepLearning)
	imputed.SocialDays = decayBehaviorInt(previous.SocialDays)
	imputed.StressScore = imputeSubjectiveInt(previous.StressScore, neutralStressScore, missedWeeks)
	return imputed
}

func decayBehaviorInt(value int) int {
	return decayInt(value, behaviorDecayRate)
}

func imputeStableFloat(value, baseline float64, missedWeeks int) float64 {
	if missedWeeks <= stableCarryWeeks {
		return value
	}
	return driftFloat(value, baseline, stableDriftRate)
}

func imputeStableInt(value int, baseline float64, missedWeeks int) int {
	if missedWeeks <= stableCarryWeeks {
		return value
	}
	return driftInt(value, baseline, stableDriftRate)
}

func imputeSubjectiveFloat(value, neutral float64, missedWeeks int) float64 {
	if missedWeeks <= subjectiveCarryWeeks {
		return value
	}
	return driftFloat(value, neutral, subjectiveDriftRate)
}

func imputeSubjectiveInt(value int, neutral float64, missedWeeks int) int {
	if missedWeeks <= subjectiveCarryWeeks {
		return value
	}
	return driftInt(value, neutral, subjectiveDriftRate)
}

func driftFloat(value, target, rate float64) float64 {
	return value + (target-value)*rate
}

func driftInt(value int, target, rate float64) int {
	return int(math.Round(driftFloat(float64(value), target, rate)))
}

func decayInt(value int, rate float64) int {
	if value <= 0 {
		return 0
	}

	next := float64(value) * (1 - rate)
	if next < 1 {
		return 0
	}

	return int(math.Round(next))
}
