package services

import (
	"health-balance/internal/models"
	"testing"
)

type MockDB struct {
	AllDates         []string
	UserProfile      *models.UserProfile
	HealthMap        map[string]*models.HealthMetrics
	FitnessMap       map[string]*models.FitnessMetrics
	CognitionMap     map[string]*models.CognitionMetrics
	RHRBaselineValue int
	Err              error
}

func (m *MockDB) GetAllDatesWithData() ([]string, error)       { return m.AllDates, m.Err }
func (m *MockDB) GetUserProfile() (*models.UserProfile, error) { return m.UserProfile, m.Err }
func (m *MockDB) GetRHRBaseline() (int, error)                 { return m.RHRBaselineValue, m.Err }
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
		// sleep: (80-75)*2 = 10
		// whtr: (0.48 - (80/180)) * 1000 = (0.48 - 0.444) * 1000 = 35.55
		// rhr: (60-60)*5 = 0
		// nutrition: (8-7)*5 = 5
		// Total: ~50.55
		score := CalculateHealthPillar(m, 60, 80.0/180.0)
		if score <= 0 {
			t.Errorf("Expected positive health score, got %f", score)
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
			date1: {VO2Max: 40, WeeklyWorkouts: 3, DailySteps: 8000, WeeklyMobility: 3, CardioRecovery: 20},
			date2: {VO2Max: 42, WeeklyWorkouts: 4, DailySteps: 10000, WeeklyMobility: 3, CardioRecovery: 25},
		},
		CognitionMap: map[string]*models.CognitionMetrics{
			date1: {DualNBackLevel: 2, ReactionTime: 250, WeeklyMindfulness: 3},
			date2: {DualNBackLevel: 3, ReactionTime: 240, WeeklyMindfulness: 4},
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

	// Verify compounding: Better stats in week 2 should result in higher score than week 1
	if scores[1].Score <= scores[0].Score {
		t.Errorf("Compounding Error: Week 2 (%f) should be higher than Week 1 (%f)", scores[1].Score, scores[0].Score)
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
