package database

import (
	"path/filepath"
	"slices"
	"testing"

	"health-balance/internal/models"
	"health-balance/internal/utils"
)

func TestGetAllDatesWithData(t *testing.T) {
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

	testDate := utils.GetPreviousSundayDate()
	healthMetrics := models.HealthMetrics{
		SleepScore:     80,
		WaistCm:        85.0,
		RHR:            60,
		NutritionScore: 7.5,
	}
	if err := db.SaveHealthMetrics(healthMetrics); err != nil {
		t.Fatalf("Failed to save health metrics: %v", err)
	}

	dates, err := db.GetAllDatesWithData()
	if err != nil {
		t.Fatalf("Failed to get all dates: %v", err)
	}

	if len(dates) == 0 {
		t.Error("Expected at least one date, got none")
	}

	found := slices.Contains(dates, testDate)
	if !found {
		t.Errorf("Expected to find test date %s in results, but it wasn't there. Available dates: %v", testDate, dates)
	}
}

func TestGetRecentHealthMetrics(t *testing.T) {
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

	// Insert test data
	healthMetrics := models.HealthMetrics{
		SleepScore:     80,
		WaistCm:        85.0,
		RHR:            60,
		NutritionScore: 7.5,
	}
	if err := db.SaveHealthMetrics(healthMetrics); err != nil {
		t.Fatalf("Failed to save health metrics: %v", err)
	}

	metrics, err := db.GetRecentHealthMetrics(5)
	if err != nil {
		t.Fatalf("Failed to get recent health metrics: %v", err)
	}

	// Since we just saved data for the current week, it shouldn't appear in recent metrics
	// (recent metrics exclude the current week)
	if len(metrics) != 0 {
		t.Errorf("Expected no recent metrics (current week excluded), got %d", len(metrics))
	}
}

func TestGetRecentFitnessMetrics(t *testing.T) {
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

	// Insert test data
	fitnessMetrics := models.FitnessMetrics{
		VO2Max:         45.0,
		Workouts:       4,
		DailySteps:     10000,
		Mobility:       3,
		CardioRecovery: 25,
	}
	if err := db.SaveFitnessMetrics(fitnessMetrics); err != nil {
		t.Fatalf("Failed to save fitness metrics: %v", err)
	}

	metrics, err := db.GetRecentFitnessMetrics(5)
	if err != nil {
		t.Fatalf("Failed to get recent fitness metrics: %v", err)
	}

	// Since we just saved data for the current week, it shouldn't appear in recent metrics
	// (recent metrics exclude the current week)
	if len(metrics) != 0 {
		t.Errorf("Expected no recent metrics (current week excluded), got %d", len(metrics))
	}
}

func TestGetRecentCognitionMetrics(t *testing.T) {
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

	// Insert test data
	cognitionMetrics := models.CognitionMetrics{
		DualNBackLevel: 3,
		ReactionTime:   240,
		Mindfulness:    4,
		DeepLearning:   60,
	}
	if err := db.SaveCognitionMetrics(cognitionMetrics); err != nil {
		t.Fatalf("Failed to save cognition metrics: %v", err)
	}

	metrics, err := db.GetRecentCognitionMetrics(5)
	if err != nil {
		t.Fatalf("Failed to get recent cognition metrics: %v", err)
	}

	// Since we just saved data for the current week, it shouldn't appear in recent metrics
	// (recent metrics exclude the current week)
	if len(metrics) != 0 {
		t.Errorf("Expected no recent metrics (current week excluded), got %d", len(metrics))
	}
}

func TestSaveAndRetrieveHealthMetrics(t *testing.T) {
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

	// Get the date for this week's Sunday
	testDate := utils.GetPreviousSundayDate()

	healthMetrics := models.HealthMetrics{
		SleepScore:     85,
		WaistCm:        82.5,
		RHR:            58,
		NutritionScore: 8.0,
	}

	if err := db.SaveHealthMetrics(healthMetrics); err != nil {
		t.Fatalf("Failed to save health metrics: %v", err)
	}

	retrieved, err := db.GetHealthMetricsByDate(testDate)
	if err != nil {
		t.Fatalf("Failed to retrieve health metrics: %v", err)
	}

	if retrieved.SleepScore != 85 {
		t.Errorf("Expected SleepScore 85, got %d", retrieved.SleepScore)
	}
	if retrieved.WaistCm != 82.5 {
		t.Errorf("Expected WaistCm 82.5, got %f", retrieved.WaistCm)
	}
	if retrieved.RHR != 58 {
		t.Errorf("Expected RHR 58, got %d", retrieved.RHR)
	}
	if retrieved.NutritionScore != 8.0 {
		t.Errorf("Expected NutritionScore 8.0, got %f", retrieved.NutritionScore)
	}
}

func TestSaveAndRetrieveFitnessMetrics(t *testing.T) {
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

	// Get the date for this week's Sunday
	testDate := utils.GetPreviousSundayDate()

	fitnessMetrics := models.FitnessMetrics{
		VO2Max:         48.0,
		Workouts:       5,
		DailySteps:     12000,
		Mobility:       4,
		CardioRecovery: 30,
	}

	if err := db.SaveFitnessMetrics(fitnessMetrics); err != nil {
		t.Fatalf("Failed to save fitness metrics: %v", err)
	}

	retrieved, err := db.GetFitnessMetricsByDate(testDate)
	if err != nil {
		t.Fatalf("Failed to retrieve fitness metrics: %v", err)
	}

	if retrieved.VO2Max != 48.0 {
		t.Errorf("Expected VO2Max 48.0, got %f", retrieved.VO2Max)
	}
	if retrieved.Workouts != 5 {
		t.Errorf("Expected Workouts 5, got %d", retrieved.Workouts)
	}
	if retrieved.DailySteps != 12000 {
		t.Errorf("Expected DailySteps 12000, got %d", retrieved.DailySteps)
	}
	if retrieved.Mobility != 4 {
		t.Errorf("Expected Mobility 4, got %d", retrieved.Mobility)
	}
	if retrieved.CardioRecovery != 30 {
		t.Errorf("Expected CardioRecovery 30, got %d", retrieved.CardioRecovery)
	}
}

func TestSaveAndRetrieveCognitionMetrics(t *testing.T) {
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

	// Get the date for this week's Sunday
	testDate := utils.GetPreviousSundayDate()

	cognitionMetrics := models.CognitionMetrics{
		DualNBackLevel: 4,
		ReactionTime:   220,
		Mindfulness:    5,
		DeepLearning:   90,
	}

	if err := db.SaveCognitionMetrics(cognitionMetrics); err != nil {
		t.Fatalf("Failed to save cognition metrics: %v", err)
	}

	retrieved, err := db.GetCognitionMetricsByDate(testDate)
	if err != nil {
		t.Fatalf("Failed to retrieve cognition metrics: %v", err)
	}

	if retrieved.DualNBackLevel != 4 {
		t.Errorf("Expected DualNBackLevel 4, got %d", retrieved.DualNBackLevel)
	}
	if retrieved.ReactionTime != 220 {
		t.Errorf("Expected ReactionTime 220, got %d", retrieved.ReactionTime)
	}
	if retrieved.Mindfulness != 5 {
		t.Errorf("Expected Mindfulness 5, got %d", retrieved.Mindfulness)
	}
	if retrieved.DeepLearning != 90 {
		t.Errorf("Expected DeepLearning 90, got %d", retrieved.DeepLearning)
	}
}

func TestDeleteMetrics(t *testing.T) {
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

	// Get the date for this week's Sunday
	testDate := utils.GetPreviousSundayDate()

	// Insert health metrics
	healthMetrics := models.HealthMetrics{
		SleepScore:     80,
		WaistCm:        85.0,
		RHR:            60,
		NutritionScore: 7.5,
	}
	if err := db.SaveHealthMetrics(healthMetrics); err != nil {
		t.Fatalf("Failed to save health metrics: %v", err)
	}

	// Verify it was saved
	_, err = db.GetHealthMetricsByDate(testDate)
	if err != nil {
		t.Fatalf("Failed to retrieve health metrics: %v", err)
	}

	// Delete the metrics
	if err := db.DeleteHealthMetrics(testDate); err != nil {
		t.Fatalf("Failed to delete health metrics: %v", err)
	}

	// Verify it was deleted
	_, err = db.GetHealthMetricsByDate(testDate)
	if err == nil {
		t.Error("Expected error when retrieving deleted health metrics, but got none")
	}
}

func TestGetRHRBaseline(t *testing.T) {
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

	// Insert health metrics with RHR
	healthMetrics := models.HealthMetrics{
		SleepScore:     80,
		WaistCm:        85.0,
		RHR:            60,
		NutritionScore: 7.5,
	}
	if err := db.SaveHealthMetrics(healthMetrics); err != nil {
		t.Fatalf("Failed to save health metrics: %v", err)
	}

	baseline, err := db.GetRHRBaseline()
	if err != nil {
		t.Fatalf("Failed to get RHR baseline: %v", err)
	}

	// Since we only have one entry, the baseline should be the RHR value
	if baseline != 60 {
		t.Errorf("Expected RHR baseline 60, got %d", baseline)
	}
}

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

func TestPushSubscriptions(t *testing.T) {
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

	sub := models.PushSubscription{
		Endpoint:     "https://example.com/subscription",
		P256dh:       "p256dh_key",
		Auth:         "auth_key",
		ReminderDay:  1,
		ReminderTime: "09:00",
		Timezone:     "UTC",
	}

	if err := db.SavePushSubscription(sub); err != nil {
		t.Fatalf("Failed to save push subscription: %v", err)
	}

	retrieved, err := db.GetAnyPushSubscription()
	if err != nil {
		t.Fatalf("Failed to get push subscription: %v", err)
	}

	if retrieved.Endpoint != "https://example.com/subscription" {
		t.Errorf("Expected Endpoint 'https://example.com/subscription', got '%s'", retrieved.Endpoint)
	}
	if retrieved.P256dh != "p256dh_key" {
		t.Errorf("Expected P256dh 'p256dh_key', got '%s'", retrieved.P256dh)
	}
	if retrieved.Auth != "auth_key" {
		t.Errorf("Expected Auth 'auth_key', got '%s'", retrieved.Auth)
	}
	if retrieved.ReminderDay != 1 {
		t.Errorf("Expected ReminderDay 1, got %d", retrieved.ReminderDay)
	}
	if retrieved.ReminderTime != "09:00" {
		t.Errorf("Expected ReminderTime '09:00', got '%s'", retrieved.ReminderTime)
	}
	if retrieved.Timezone != "UTC" {
		t.Errorf("Expected Timezone 'UTC', got '%s'", retrieved.Timezone)
	}
}
