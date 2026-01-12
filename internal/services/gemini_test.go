package services

import (
	"os"
	"testing"
	"time"

	"health-balance/internal/models"
	"health-balance/internal/testutil"
)

func TestConstructPrompt(t *testing.T) {
	profile := &models.UserProfile{
		Sex:       "male",
		HeightCm:  180.0,
		BirthDate: "1990-01-01",
	}

	now := time.Now()
	date1 := now.AddDate(0, 0, -7).Format("2006-01-02")  // 1 week ago
	date2 := now.AddDate(0, 0, -14).Format("2006-01-02") // 2 weeks ago

	data := []WeeklyData{
		{
			Score: models.MasterScore{
				Date:           date1,
				Score:          1050.5,
				HealthScore:    60.0,
				FitnessScore:   40.0,
				CognitionScore: 20.0,
				AgingTax:       10.0,
			},
			Health: &models.HealthMetrics{
				SleepScore:     80,
				WaistCm:        85.0,
				RHR:            60,
				NutritionScore: 8.0,
			},
			Fitness: &models.FitnessMetrics{
				VO2Max:         45.0,
				Workouts:       4,
				DailySteps:     10000,
				Mobility:       3,
				CardioRecovery: 25,
			},
			Cognition: &models.CognitionMetrics{
				DualNBackLevel: 3,
				ReactionTime:   240,
				Mindfulness:    4,
				DeepLearning:   40,
			},
		},
		{
			Score: models.MasterScore{
				Date:           date2,
				Score:          1040.0,
				HealthScore:    55.0,
				FitnessScore:   35.0,
				CognitionScore: 15.0,
				AgingTax:       12.0,
			},
			Health: &models.HealthMetrics{
				SleepScore:     75,
				WaistCm:        86.0,
				RHR:            62,
				NutritionScore: 7.0,
			},
			Fitness: &models.FitnessMetrics{
				VO2Max:         42.0,
				Workouts:       3,
				DailySteps:     8000,
				Mobility:       2,
				CardioRecovery: 20,
			},
			Cognition: &models.CognitionMetrics{
				DualNBackLevel: 2,
				ReactionTime:   250,
				Mindfulness:    3,
				DeepLearning:   30,
			},
		},
	}

	prompt := constructPrompt(profile, data)

	if prompt == "" {
		t.Error("constructPrompt returned an empty string")
	}

	expectedElements := []string{
		"expert longevity and health coach",
		"180.0 cm",
		"male",
		"Health Pillar",
		"Fitness Pillar",
		"Cognition Pillar",
		date1,
		date2,
		"Total: 1050.5",
		"Sleep Score: 80",
		"VO2 Max: 45.0",
		"Dual N-Back Level: 3",
		"Reaction Time: 240",
	}

	for _, element := range expectedElements {
		if element == "expert longevity and health coach" && len(prompt) < 100 {
			t.Error("Prompt appears too short, might be missing content")
		}
	}
}

func TestConstructPromptWithEmptyData(t *testing.T) {
	profile := &models.UserProfile{
		Sex:       "female",
		HeightCm:  165.0,
		BirthDate: "1985-05-15",
	}

	data := []WeeklyData{}

	prompt := constructPrompt(profile, data)

	if prompt == "" {
		t.Error("constructPrompt returned an empty string for empty data")
	}

	// Should still contain the basic structure
	if len(prompt) < 50 {
		t.Error("Prompt appears too short even with empty data")
	}
}

func TestGetHealthSummaryWithoutApiKey(t *testing.T) {
	oldApiKey := os.Getenv("GEMINI_API_KEY")
	_ = os.Unsetenv("GEMINI_API_KEY")
	defer func() {
		_ = os.Setenv("GEMINI_API_KEY", oldApiKey)
	}()

	mockDB := &testutil.MockDB{
		GetUserProfileFunc: func() (*models.UserProfile, error) {
			return &models.UserProfile{
				BirthDate: "1990-01-01",
				Sex:       "male",
				HeightCm:  180.0,
			}, nil
		},
		GetAllDatesWithDataFunc: func() ([]string, error) {
			return []string{"2023-01-01"}, nil
		},
	}

	_, err := GetHealthSummary(mockDB)
	if err == nil {
		t.Error("Expected error when GEMINI_API_KEY is not set, but got none")
	}

	if err.Error() != "GEMINI_API_KEY environment variable is not set" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestGetHealthSummaryWithoutProfile(t *testing.T) {
	oldApiKey := os.Getenv("GEMINI_API_KEY")
	_ = os.Setenv("GEMINI_API_KEY", "fake-key")
	defer func() {
		_ = os.Setenv("GEMINI_API_KEY", oldApiKey)
	}()

	mockDB := &testutil.MockDB{
		GetUserProfileFunc: func() (*models.UserProfile, error) {
			return nil, nil // No profile
		},
	}

	_, err := GetHealthSummary(mockDB)
	if err == nil {
		t.Error("Expected error when user profile is not available, but got none")
	}

	if err.Error() != "user profile required for summary" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}
