package services

import (
	"health-balance/internal/models"
	"testing"
)

func TestConstructPrompt(t *testing.T) {
	profile := &models.UserProfile{
		Sex:       "male",
		HeightCm:  180,
		BirthDate: "1990-01-01",
	}

	data := []WeeklyData{
		{
			Score: models.MasterScore{
				Date:           "2023-10-01",
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
				WeeklyWorkouts: 4,
				DailySteps:     10000,
				WeeklyMobility: 3,
				CardioRecovery: 25,
			},
			Cognition: &models.CognitionMetrics{
				DualNBackLevel:    3,
				ReactionTime:      240,
				WeeklyMindfulness: 4,
			},
		},
	}

	prompt := constructPrompt(profile, data)

	if prompt == "" {
		t.Errorf("ConstructPrompt returned an empty prompt")
	}

	expectedParts := []string{
		"longevity and health coach",
		"180.0 cm",
		"male",
		"Health Pillar",
		"Fitness Pillar",
		"Cognition Pillar",
		"2023-10-01",
		"1050.5",
		"Sleep Score: 80",
		"VO2 Max: 45.0",
		"Reaction Time: 240",
	}

	for _, part := range expectedParts {
		if !contains(prompt, part) {
			t.Errorf("Prompt missing expected part: %s", part)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s[:len(substr)] == substr || (len(s) > len(substr) && contains(s[1:], substr)))
}
