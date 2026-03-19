package services

import (
	"health-balance/internal/models"
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
		m := models.HealthMetrics{SleepScore: 80, WaistCm: 80, RHR: 60, NutritionScore: 8}
		score := CalculateHealthPillar(m, 60, 80.0/180.0)
		if score <= 0 {
			t.Errorf("Expected positive health score, got %f", score)
		}
	})

	t.Run("Fitness Pillar Caps Excess Workout Volume", func(t *testing.T) {
		high := models.FitnessMetrics{VO2Max: 42, Workouts: 7, DailySteps: 9000, Mobility: 3, CardioRecovery: 25}
		extreme := models.FitnessMetrics{VO2Max: 42, Workouts: 12, DailySteps: 9000, Mobility: 3, CardioRecovery: 25}

		highScore := CalculateFitnessPillar(high, 40)
		extremeScore := CalculateFitnessPillar(extreme, 40)

		if highScore != extremeScore {
			t.Errorf("Expected workout contribution to cap out, got %.2f vs %.2f", highScore, extremeScore)
		}
	})

	t.Run("Reserve Markers Outweigh One Behavior Spike", func(t *testing.T) {
		reserveHeavy := models.FitnessMetrics{VO2Max: 46, Workouts: 4, DailySteps: 9000, Mobility: 3, CardioRecovery: 25}
		behaviorHeavy := models.FitnessMetrics{VO2Max: 42, Workouts: 10, DailySteps: 9000, Mobility: 3, CardioRecovery: 25}

		reserveScore := CalculateFitnessPillar(reserveHeavy, 40)
		behaviorScore := CalculateFitnessPillar(behaviorHeavy, 40)

		if reserveScore <= behaviorScore {
			t.Errorf("Expected stronger VO2 reserve to outweigh excess workouts, got reserve %.2f vs behavior %.2f", reserveScore, behaviorScore)
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
	date1 := "2025-12-01"
	date2 := "2025-12-08"

	mock := &MockDB{
		AllDates:    []string{date2, date1},
		UserProfile: &models.UserProfile{BirthDate: "1990-12-26", HeightCm: 180, Sex: "male"},
		HealthMap: map[string]*models.HealthMetrics{
			date1: {RHR: 65, WaistCm: 85, SleepScore: 75, NutritionScore: 7},
			date2: {RHR: 60, WaistCm: 85, SleepScore: 85, NutritionScore: 8},
		},
		FitnessMap: map[string]*models.FitnessMetrics{
			date1: {VO2Max: 40, Workouts: 3, DailySteps: 8000, Mobility: 3, CardioRecovery: 20},
			date2: {VO2Max: 42, Workouts: 4, DailySteps: 10000, Mobility: 3, CardioRecovery: 25},
		},
		CognitionMap: map[string]*models.CognitionMetrics{
			date1: {DualNBackLevel: 2, ReactionTime: 250, Mindfulness: 3, DeepLearning: 50},
			date2: {DualNBackLevel: 3, ReactionTime: 240, Mindfulness: 4, DeepLearning: 80},
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
	health := models.HealthMetrics{SleepScore: 84, WaistCm: 82, RHR: 58, NutritionScore: 8.5}
	fitness := models.FitnessMetrics{VO2Max: 47, Workouts: 5, DailySteps: 10500, Mobility: 4, CardioRecovery: 28}
	cognition := models.CognitionMetrics{DualNBackLevel: 3, ReactionTime: 225, Mindfulness: 4, DeepLearning: 120}

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
			240,
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
	date1 := "2025-01-05"
	date2 := "2025-04-06"

	mock := &MockDB{
		AllDates:    []string{date2, date1},
		UserProfile: &models.UserProfile{BirthDate: "1990-01-01", HeightCm: 180, Sex: "male"},
		HealthMap: map[string]*models.HealthMetrics{
			date1: {RHR: 70, WaistCm: 85, SleepScore: 75, NutritionScore: 7},
			date2: {RHR: 60, WaistCm: 85, SleepScore: 75, NutritionScore: 7},
		},
		FitnessMap: map[string]*models.FitnessMetrics{
			date1: {VO2Max: 42, Workouts: 3, DailySteps: 8000, Mobility: 3, CardioRecovery: 25},
			date2: {VO2Max: 42, Workouts: 3, DailySteps: 8000, Mobility: 3, CardioRecovery: 25},
		},
		CognitionMap: map[string]*models.CognitionMetrics{
			date1: {DualNBackLevel: 2, ReactionTime: 240, Mindfulness: 3, DeepLearning: 90},
			date2: {DualNBackLevel: 2, ReactionTime: 240, Mindfulness: 3, DeepLearning: 90},
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
	dates := []string{"2025-01-26", "2025-01-19", "2025-01-12", "2025-01-05"}

	buildMock := func(workouts []int) *MockDB {
		healthMap := make(map[string]*models.HealthMetrics, len(dates))
		fitnessMap := make(map[string]*models.FitnessMetrics, len(dates))
		cognitionMap := make(map[string]*models.CognitionMetrics, len(dates))
		rhrBaselineByDate := make(map[string]int, len(dates))

		for i, date := range []string{"2025-01-05", "2025-01-12", "2025-01-19", "2025-01-26"} {
			healthMap[date] = &models.HealthMetrics{RHR: 60, WaistCm: 85, SleepScore: 80, NutritionScore: 8}
			fitnessMap[date] = &models.FitnessMetrics{
				VO2Max:         42,
				Workouts:       workouts[i],
				DailySteps:     8000,
				Mobility:       3,
				CardioRecovery: 25,
			}
			cognitionMap[date] = &models.CognitionMetrics{DualNBackLevel: 2, ReactionTime: 240, Mindfulness: 3, DeepLearning: 90}
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
	mock := &MockDB{
		AllDates:    []string{"2025-12-26"},
		UserProfile: &models.UserProfile{BirthDate: "1995-12-26", HeightCm: 180},
		HealthMap: map[string]*models.HealthMetrics{
			"2025-12-26": {SleepScore: 80},
		},
		FitnessMap: map[string]*models.FitnessMetrics{
			"2025-12-26": {VO2Max: 40},
		},
		CognitionMap: make(map[string]*models.CognitionMetrics), // MISSING
	}

	scores, _ := GetAllWeeklyScores(mock)
	if len(scores) != 0 {
		t.Errorf("Data Gate failed: Should skip weeks with missing pillars, got %d scores", len(scores))
	}
}
