package services

import (
	"health-balance/internal/models"
	"health-balance/internal/utils"
	"testing"
	"time"
)

type MockDB struct {
	AllDates          []string
	UserProfile       *models.UserProfile
	HealthMap         map[string]*models.HealthMetrics
	FitnessMap        map[string]*models.FitnessMetrics
	CognitionMap      map[string]*models.CognitionMetrics
	RHRBaselineValue  int
	RHRBaselineByDate map[string]int
	Err               error
}

func (m *MockDB) GetAllDatesWithData() ([]string, error)       { return m.AllDates, m.Err }
func (m *MockDB) GetUserProfile() (*models.UserProfile, error) { return m.UserProfile, m.Err }
func (m *MockDB) GetRHRBaselineForDate(date string) (int, error) {
	if m.RHRBaselineByDate != nil {
		if baseline, ok := m.RHRBaselineByDate[date]; ok {
			return baseline, m.Err
		}
	}
	return m.RHRBaselineValue, m.Err
}
func (m *MockDB) GetHealthMetricsByDate(d string) (*models.HealthMetrics, error) {
	return m.HealthMap[d], m.Err
}
func (m *MockDB) GetFitnessMetricsByDate(d string) (*models.FitnessMetrics, error) {
	return m.FitnessMap[d], m.Err
}
func (m *MockDB) GetCognitionMetricsByDate(d string) (*models.CognitionMetrics, error) {
	return m.CognitionMap[d], m.Err
}
func (m *MockDB) GetRecentHealthMetrics(l int) ([]models.HealthMetrics, error)       { return nil, nil }
func (m *MockDB) GetRecentFitnessMetrics(l int) ([]models.FitnessMetrics, error)     { return nil, nil }
func (m *MockDB) GetRecentCognitionMetrics(l int) ([]models.CognitionMetrics, error) { return nil, nil }
func (m *MockDB) SaveHealthMetrics(h models.HealthMetrics) error                     { return nil }
func (m *MockDB) SaveFitnessMetrics(f models.FitnessMetrics) error                   { return nil }
func (m *MockDB) SaveCognitionMetrics(c models.CognitionMetrics) error               { return nil }
func (m *MockDB) SaveUserProfile(p models.UserProfile) error                         { return nil }
func (m *MockDB) SavePushSubscription(sub models.PushSubscription) error             { return nil }
func (m *MockDB) GetAllSubscriptions() ([]models.PushSubscription, error) {
	return nil, nil
}
func (m *MockDB) GetAnyPushSubscription() (*models.PushSubscription, error) { return nil, nil }
func (m *MockDB) DeletePushSubscription(endpoint string) error              { return nil }
func (m *MockDB) DeleteHealthMetrics(date string) error                     { return nil }
func (m *MockDB) DeleteFitnessMetrics(date string) error                    { return nil }
func (m *MockDB) DeleteCognitionMetrics(date string) error                  { return nil }
func (m *MockDB) Close() error                                              { return nil }

func TestCalculatePillars(t *testing.T) {
	t.Run("Health Pillar Math", func(t *testing.T) {
		m := models.HealthMetrics{SleepScore: 80, WaistCm: 80, RHR: 60, SystolicBP: 118, DiastolicBP: 76, NutritionScore: 8}
		score := CalculateHealthPillar(m, 60, 80.0/180.0)
		if score <= 0 {
			t.Errorf("Expected positive health score, got %f", score)
		}
	})

	t.Run("Fitness Pillar Caps Excess Workout Volume", func(t *testing.T) {
		high := models.FitnessMetrics{VO2Max: 42, Workouts: 7, DailySteps: 9000, Mobility: 3, CardioRecovery: 25}
		extreme := models.FitnessMetrics{VO2Max: 42, Workouts: 12, DailySteps: 9000, Mobility: 3, CardioRecovery: 25}

		highScore := CalculateFitnessPillar(high, 40, 75)
		extremeScore := CalculateFitnessPillar(extreme, 40, 75)

		if highScore != extremeScore {
			t.Errorf("Expected workout contribution to cap out, got %.2f vs %.2f", highScore, extremeScore)
		}
	})

	t.Run("Reserve Markers Outweigh One Behavior Spike", func(t *testing.T) {
		reserveHeavy := models.FitnessMetrics{VO2Max: 46, Workouts: 4, DailySteps: 9000, Mobility: 3, CardioRecovery: 25}
		behaviorHeavy := models.FitnessMetrics{VO2Max: 42, Workouts: 10, DailySteps: 9000, Mobility: 3, CardioRecovery: 25}

		reserveScore := CalculateFitnessPillar(reserveHeavy, 40, 75)
		behaviorScore := CalculateFitnessPillar(behaviorHeavy, 40, 75)

		if reserveScore <= behaviorScore {
			t.Errorf("Expected stronger VO2 reserve to outweigh excess workouts, got reserve %.2f vs behavior %.2f", reserveScore, behaviorScore)
		}
	})

	t.Run("Blood Pressure Rewards Healthier Range", func(t *testing.T) {
		healthy := calculateBloodPressurePoints(models.HealthMetrics{SystolicBP: 118, DiastolicBP: 76})
		elevated := calculateBloodPressurePoints(models.HealthMetrics{SystolicBP: 136, DiastolicBP: 88})

		if healthy <= elevated {
			t.Fatalf("Expected healthier blood pressure to score better, got %.2f vs %.2f", healthy, elevated)
		}
	})

	t.Run("Strength Scores Leg Press Performance with RSI", func(t *testing.T) {
		// 60kg person pressing 180kg for 10 reps: RSI = (180/60) * 10 = 30 (strong)
		lightPerson := calculateLowerBodyStrengthPoints(models.FitnessMetrics{LowerBodyWeight: 180, LowerBodyReps: 10}, 60)
		// 90kg person pressing 180kg for 10 reps: RSI = (180/90) * 10 = 20 (moderate)
		heavyPerson := calculateLowerBodyStrengthPoints(models.FitnessMetrics{LowerBodyWeight: 180, LowerBodyReps: 10}, 90)

		if lightPerson <= 0 {
			t.Fatalf("Expected leg press strength score to be positive, got %.2f", lightPerson)
		}
		if lightPerson <= heavyPerson {
			t.Fatalf("Expected lighter person with same absolute weight to score better (RSI), got %.2f vs %.2f", lightPerson, heavyPerson)
		}
	})

	t.Run("Behavior Smoothing Uses Recent Consistency", func(t *testing.T) {
		history := []models.FitnessMetrics{
			{Workouts: 0, DailySteps: 8000, Mobility: 3},
			{Workouts: 0, DailySteps: 8000, Mobility: 3},
			{Workouts: 0, DailySteps: 8000, Mobility: 3},
			{Workouts: 8, DailySteps: 8000, Mobility: 3},
		}

		smoothed := smoothFitnessBehaviors(history, history[len(history)-1])
		if smoothed.Workouts != 2 {
			t.Fatalf("Expected workout smoothing to average recent weeks, got %d", smoothed.Workouts)
		}
	})
}

func TestGetAllWeeklyScores_Compounding(t *testing.T) {
	currentWeek, err := time.Parse("2006-01-02", utils.GetCurrentWeekSundayDate())
	if err != nil {
		t.Fatalf("Failed to parse current week: %v", err)
	}

	date1 := currentWeek.AddDate(0, 0, -7).Format("2006-01-02")
	date2 := currentWeek.Format("2006-01-02")

	mock := &MockDB{
		AllDates:    []string{date2, date1},
		UserProfile: &models.UserProfile{BirthDate: "1990-12-26", HeightCm: 180, Sex: "male"},
		HealthMap: map[string]*models.HealthMetrics{
			date1: {RHR: 65, WaistCm: 85, BodyWeightKg: 75, SleepScore: 75, NutritionScore: 7},
			date2: {RHR: 60, WaistCm: 85, BodyWeightKg: 75, SleepScore: 85, NutritionScore: 8},
		},
		FitnessMap: map[string]*models.FitnessMetrics{
			date1: {VO2Max: 40, Workouts: 3, DailySteps: 8000, Mobility: 3, CardioRecovery: 20, DeadHangSeconds: 50},
			date2: {VO2Max: 42, Workouts: 4, DailySteps: 10000, Mobility: 3, CardioRecovery: 25, DeadHangSeconds: 65},
		},
		CognitionMap: map[string]*models.CognitionMetrics{
			date1: {Mindfulness: 3, DeepLearning: 50, StressScore: 3, SocialDays: 3},
			date2: {Mindfulness: 4, DeepLearning: 80, StressScore: 2, SocialDays: 5},
		},
		RHRBaselineValue: 65,
	}

	scores, err := GetAllWeeklyScores(mock)
	if err != nil {
		t.Fatalf("Failed to calculate: %v", err)
	}

	if len(scores) != 2 {
		t.Fatalf("Expected 2 weeks of scores, got %d", len(scores))
	}

	// Verify chronological processing: date1 must be first in result slice
	if scores[0].Date != date1 {
		t.Errorf("Ordering mismatch: Index 0 should be %s, got %s", date1, scores[0].Date)
	}

	// Better later inputs should still improve the pillar signals, even if the
	// slow-moving total score does not jump immediately.
	if scores[1].HealthScore <= scores[0].HealthScore {
		t.Errorf("Expected Week 2 health pillar (%f) to exceed Week 1 (%f)", scores[1].HealthScore, scores[0].HealthScore)
	}
	if scores[1].FitnessScore <= scores[0].FitnessScore {
		t.Errorf("Expected Week 2 fitness pillar (%f) to exceed Week 1 (%f)", scores[1].FitnessScore, scores[0].FitnessScore)
	}
	if scores[1].CognitionScore <= scores[0].CognitionScore {
		t.Errorf("Expected Week 2 cognition pillar (%f) to exceed Week 1 (%f)", scores[1].CognitionScore, scores[0].CognitionScore)
	}
}

func TestCalculateMasterScore_ConvergesInsteadOfRunningAway(t *testing.T) {
	profile := models.UserProfile{BirthDate: "1990-01-01", HeightCm: 180, Sex: "male"}
	health := models.HealthMetrics{SleepScore: 84, WaistCm: 82, BodyWeightKg: 75, RHR: 58, NutritionScore: 8.5}
	fitness := models.FitnessMetrics{VO2Max: 47, Workouts: 5, DailySteps: 10500, Mobility: 4, CardioRecovery: 28, DeadHangSeconds: 85}
	cognition := models.CognitionMetrics{Mindfulness: 4, DeepLearning: 120, StressScore: 2, SocialDays: 5}

	score := defaultMasterScore
	calculationDate := time.Date(2026, time.January, 4, 0, 0, 0, 0, time.UTC)

	for range 26 {
		nextScore, _, _, _, _ := CalculateMasterScore(
			score,
			profile,
			health,
			fitness,
			cognition,
			65,
			42,
			health.WaistCm/profile.HeightCm,
			calculationDate,
		)
		score = nextScore
		calculationDate = calculationDate.AddDate(0, 0, 7)
	}

	if score <= defaultMasterScore {
		t.Fatalf("Expected strong long-term metrics to improve the score, got %.2f", score)
	}

	if score >= 1100 {
		t.Fatalf("Expected the score to converge instead of running away, got %.2f", score)
	}
}

func TestGetAllWeeklyScores_UsesHistoricalRHRBaseline(t *testing.T) {
	currentWeek, err := time.Parse("2006-01-02", utils.GetCurrentWeekSundayDate())
	if err != nil {
		t.Fatalf("Failed to parse current week: %v", err)
	}

	date1 := currentWeek.AddDate(0, 0, -7).Format("2006-01-02")
	date2 := currentWeek.Format("2006-01-02")

	mock := &MockDB{
		AllDates:    []string{date2, date1},
		UserProfile: &models.UserProfile{BirthDate: "1990-01-01", HeightCm: 180, Sex: "male"},
		HealthMap: map[string]*models.HealthMetrics{
			date1: {RHR: 70, WaistCm: 85, BodyWeightKg: 75, SleepScore: 75, NutritionScore: 7},
			date2: {RHR: 60, WaistCm: 85, BodyWeightKg: 75, SleepScore: 75, NutritionScore: 7},
		},
		FitnessMap: map[string]*models.FitnessMetrics{
			date1: {VO2Max: 42, Workouts: 3, DailySteps: 8000, Mobility: 3, CardioRecovery: 25, DeadHangSeconds: 60},
			date2: {VO2Max: 42, Workouts: 3, DailySteps: 8000, Mobility: 3, CardioRecovery: 25, DeadHangSeconds: 60},
		},
		CognitionMap: map[string]*models.CognitionMetrics{
			date1: {Mindfulness: 3, DeepLearning: 90, StressScore: 3, SocialDays: 4},
			date2: {Mindfulness: 3, DeepLearning: 90, StressScore: 3, SocialDays: 4},
		},
		RHRBaselineByDate: map[string]int{
			date1: 70,
			date2: 63,
		},
	}

	scores, err := GetAllWeeklyScores(mock)
	if err != nil {
		t.Fatalf("Failed to calculate: %v", err)
	}

	if len(scores) != 2 {
		t.Fatalf("Expected 2 weeks of scores, got %d", len(scores))
	}

	lateWeek := mock.HealthMap[date2]
	whtr := lateWeek.WaistCm / mock.UserProfile.HeightCm
	expectedHistorical := CalculateHealthPillar(*lateWeek, 63, whtr)
	expectedShared := CalculateHealthPillar(*lateWeek, 70, whtr)

	if scores[1].HealthScore != expectedHistorical {
		t.Fatalf("Expected later week health score %.2f using its historical RHR baseline, got %.2f", expectedHistorical, scores[1].HealthScore)
	}

	if scores[1].HealthScore >= expectedShared {
		t.Fatalf("Expected historical RHR baseline to reduce the later week score versus a shared baseline, got %.2f vs %.2f", scores[1].HealthScore, expectedShared)
	}
}

func TestGetAllWeeklyScores_WeightsConsistencyOverOneWeekSpike(t *testing.T) {
	currentWeek, err := time.Parse("2006-01-02", utils.GetCurrentWeekSundayDate())
	if err != nil {
		t.Fatalf("Failed to parse current week: %v", err)
	}

	ordered := []string{
		currentWeek.AddDate(0, 0, -21).Format("2006-01-02"),
		currentWeek.AddDate(0, 0, -14).Format("2006-01-02"),
		currentWeek.AddDate(0, 0, -7).Format("2006-01-02"),
		currentWeek.Format("2006-01-02"),
	}
	dates := []string{ordered[3], ordered[2], ordered[1], ordered[0]}

	buildMock := func(workouts []int) *MockDB {
		healthMap := make(map[string]*models.HealthMetrics, len(dates))
		fitnessMap := make(map[string]*models.FitnessMetrics, len(dates))
		cognitionMap := make(map[string]*models.CognitionMetrics, len(dates))
		rhrBaselineByDate := make(map[string]int, len(dates))

		for i, date := range ordered {
			healthMap[date] = &models.HealthMetrics{RHR: 60, WaistCm: 85, BodyWeightKg: 75, SleepScore: 80, NutritionScore: 8}
			fitnessMap[date] = &models.FitnessMetrics{
				VO2Max:          42,
				Workouts:        workouts[i],
				DailySteps:      8000,
				Mobility:        3,
				CardioRecovery:  25,
				DeadHangSeconds: 60,
			}
			cognitionMap[date] = &models.CognitionMetrics{Mindfulness: 3, DeepLearning: 90, StressScore: 3, SocialDays: 4}
			rhrBaselineByDate[date] = 60
		}

		return &MockDB{
			AllDates:          dates,
			UserProfile:       &models.UserProfile{BirthDate: "1990-01-01", HeightCm: 180, Sex: "male"},
			HealthMap:         healthMap,
			FitnessMap:        fitnessMap,
			CognitionMap:      cognitionMap,
			RHRBaselineByDate: rhrBaselineByDate,
		}
	}

	spikeScores, err := GetAllWeeklyScores(buildMock([]int{0, 0, 0, 8}))
	if err != nil {
		t.Fatalf("Failed to calculate spike scenario: %v", err)
	}

	consistentScores, err := GetAllWeeklyScores(buildMock([]int{4, 4, 4, 4}))
	if err != nil {
		t.Fatalf("Failed to calculate consistent scenario: %v", err)
	}

	spikeLatest := spikeScores[len(spikeScores)-1]
	consistentLatest := consistentScores[len(consistentScores)-1]

	if consistentLatest.FitnessScore <= spikeLatest.FitnessScore {
		t.Fatalf("Expected consistent training to outperform one-week spike, got %.2f vs %.2f", consistentLatest.FitnessScore, spikeLatest.FitnessScore)
	}

	if consistentLatest.Score <= spikeLatest.Score {
		t.Fatalf("Expected consistent training to produce a higher total score, got %.2f vs %.2f", consistentLatest.Score, spikeLatest.Score)
	}
}

func TestDataGate_Behavior(t *testing.T) {
	currentWeek, err := time.Parse("2006-01-02", utils.GetCurrentWeekSundayDate())
	if err != nil {
		t.Fatalf("Failed to parse current week: %v", err)
	}

	date := currentWeek.Format("2006-01-02")

	mock := &MockDB{
		AllDates:    []string{date},
		UserProfile: &models.UserProfile{BirthDate: "1995-12-26", HeightCm: 180},
		HealthMap: map[string]*models.HealthMetrics{
			date: {SleepScore: 80},
		},
		FitnessMap: map[string]*models.FitnessMetrics{
			date: {VO2Max: 40},
		},
		CognitionMap: make(map[string]*models.CognitionMetrics), // MISSING
	}

	scores, _ := GetAllWeeklyScores(mock)
	if len(scores) != 0 {
		t.Errorf("Data Gate failed: Should skip weeks with missing pillars, got %d scores", len(scores))
	}
}

func TestGetAllWeeklyScores_FillsMissingWeeksAndAppliesAging(t *testing.T) {
	currentWeek, err := time.Parse("2006-01-02", utils.GetCurrentWeekSundayDate())
	if err != nil {
		t.Fatalf("Failed to parse current week: %v", err)
	}

	date1 := currentWeek.AddDate(0, 0, -14).Format("2006-01-02")
	missingDate := currentWeek.AddDate(0, 0, -7).Format("2006-01-02")
	date3 := currentWeek.Format("2006-01-02")

	mock := &MockDB{
		AllDates:    []string{date3, date1},
		UserProfile: &models.UserProfile{BirthDate: "1990-01-01", HeightCm: 180, Sex: "male"},
		HealthMap: map[string]*models.HealthMetrics{
			date1: {RHR: 60, WaistCm: 85, BodyWeightKg: 75, SleepScore: 80, NutritionScore: 8, SystolicBP: 120, DiastolicBP: 80},
			date3: {RHR: 60, WaistCm: 85, BodyWeightKg: 75, SleepScore: 80, NutritionScore: 8, SystolicBP: 120, DiastolicBP: 80},
		},
		FitnessMap: map[string]*models.FitnessMetrics{
			date1: {VO2Max: 42, Workouts: 4, DailySteps: 8000, Mobility: 3, CardioRecovery: 25, LowerBodyWeight: 180, LowerBodyReps: 10, DeadHangSeconds: 60},
			date3: {VO2Max: 42, Workouts: 4, DailySteps: 8000, Mobility: 3, CardioRecovery: 25, LowerBodyWeight: 180, LowerBodyReps: 10, DeadHangSeconds: 60},
		},
		CognitionMap: map[string]*models.CognitionMetrics{
			date1: {Mindfulness: 3, DeepLearning: 90, StressScore: 2, SocialDays: 4},
			date3: {Mindfulness: 3, DeepLearning: 90, StressScore: 2, SocialDays: 4},
		},
		RHRBaselineByDate: map[string]int{
			date1:       60,
			missingDate: 60,
			date3:       60,
		},
	}

	scores, err := GetAllWeeklyScores(mock)
	if err != nil {
		t.Fatalf("Failed to calculate: %v", err)
	}

	if len(scores) != 3 {
		t.Fatalf("Expected 3 weekly scores including the missing week, got %d", len(scores))
	}

	if scores[1].Date != missingDate {
		t.Fatalf("Expected missing week score for %s, got %s", missingDate, scores[1].Date)
	}

	if scores[1].AgingTax <= 0 {
		t.Fatalf("Expected aging tax to apply during missing week, got %.4f", scores[1].AgingTax)
	}
}

func TestImputationRules_SubjectiveCarryThenDrift(t *testing.T) {
	start := models.HealthMetrics{SleepScore: 80, NutritionScore: 8}
	profile := models.UserProfile{HeightCm: 180}

	weekOne := imputeHealthMetrics(start, 1, profile, 60)
	if weekOne.SleepScore != 80 || weekOne.NutritionScore != 8 {
		t.Fatalf("Expected first missed week to carry subjective values, got sleep=%d nutrition=%.1f", weekOne.SleepScore, weekOne.NutritionScore)
	}

	weekTwo := imputeHealthMetrics(weekOne, 2, profile, 60)
	if weekTwo.SleepScore != 78 {
		t.Fatalf("Expected second missed week sleep to drift toward neutral, got %d", weekTwo.SleepScore)
	}
	if weekTwo.NutritionScore != 7.5 {
		t.Fatalf("Expected second missed week nutrition to drift toward neutral, got %.1f", weekTwo.NutritionScore)
	}
}

func TestImputationRules_StableCarryThenDrift(t *testing.T) {
	start := models.FitnessMetrics{VO2Max: 50, CardioRecovery: 30, LowerBodyWeight: 200, LowerBodyReps: 12, DeadHangSeconds: 90}
	profile := models.UserProfile{Sex: "male"}

	weekOne := imputeFitnessMetrics(start, 1, 35, profile)
	weekTwo := imputeFitnessMetrics(weekOne, 2, 35, profile)
	if weekTwo.VO2Max != 50 || weekTwo.CardioRecovery != 30 {
		t.Fatalf("Expected first two missed weeks to carry stable values, got vo2=%.1f recovery=%d", weekTwo.VO2Max, weekTwo.CardioRecovery)
	}

	weekThree := imputeFitnessMetrics(weekTwo, 3, 35, profile)
	if weekThree.VO2Max >= weekTwo.VO2Max {
		t.Fatalf("Expected stable metrics to drift toward baseline after two missed weeks, got %.1f vs %.1f", weekThree.VO2Max, weekTwo.VO2Max)
	}
	if weekThree.CardioRecovery != 29 {
		t.Fatalf("Expected cardio recovery to drift toward neutral, got %d", weekThree.CardioRecovery)
	}
}

func TestImputationRules_BehaviorsDecayTowardZero(t *testing.T) {
	start := models.CognitionMetrics{Mindfulness: 4, DeepLearning: 90, SocialDays: 5, StressScore: 2}

	weekOne := imputeCognitionMetrics(start, 1)
	if weekOne.Mindfulness != 2 || weekOne.DeepLearning != 45 || weekOne.SocialDays != 3 {
		t.Fatalf("Expected behaviors to decay on first missed week, got %+v", weekOne)
	}

	weekTwo := imputeCognitionMetrics(weekOne, 2)
	if weekTwo.Mindfulness != 1 || weekTwo.DeepLearning != 23 || weekTwo.SocialDays != 2 {
		t.Fatalf("Expected behaviors to keep decaying, got %+v", weekTwo)
	}
	if weekTwo.StressScore != 3 {
		t.Fatalf("Expected stress to drift toward neutral after the first missed week, got %d", weekTwo.StressScore)
	}
}
