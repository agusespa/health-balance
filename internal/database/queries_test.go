package database

import (
	"path/filepath"
	"testing"

	"health-balance/internal/models"
)

func TestUserProfile(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Error closing database: %v", err)
		}
	}()

	// Create a user profile
	profile := models.UserProfile{
		BirthDate: "1990-01-01",
		Sex:       "male",
		HeightCm:  180.0,
	}

	// Save the profile
	if err := db.SaveUserProfile(profile); err != nil {
		t.Fatalf("Failed to save user profile: %v", err)
	}

	// Retrieve the profile
	retrieved, err := db.GetUserProfile()
	if err != nil {
		t.Fatalf("Failed to get user profile: %v", err)
	}

	if retrieved.BirthDate != "1990-01-01" {
		t.Errorf("Expected BirthDate '1990-01-01', got '%s'", retrieved.BirthDate)
	}
	if retrieved.Sex != "male" {
		t.Errorf("Expected Sex 'male', got '%s'", retrieved.Sex)
	}
	if retrieved.HeightCm != 180.0 {
		t.Errorf("Expected HeightCm 180.0, got %f", retrieved.HeightCm)
	}
}
