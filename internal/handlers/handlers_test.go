package handlers

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"health-balance/internal/models"
	"health-balance/internal/testutil"
)

func setupTestHandler() (*Handler, *testutil.MockDB) {
	// Create templates with all the templates that are used in the handlers
	templates := template.Must(template.New("").Parse(`
{{define "index.html"}}<html><body>{{if .CurrentScore}}{{.CurrentScore.Score}}{{end}}</body></html>{{end}}
{{define "settings.html"}}<html><body>Settings</body></html>{{end}}
{{define "rationale.html"}}<html><body>Rationale</body></html>{{end}}
{{define "scores.html"}}<html><body>Scores</body></html>{{end}}
{{define "health_metrics.html"}}<html><body>Health Metrics</body></html>{{end}}
{{define "fitness_metrics.html"}}<html><body>Fitness Metrics</body></html>{{end}}
{{define "cognition_metrics.html"}}<html><body>Cognition Metrics</body></html>{{end}}
{{define "score_display"}}<html><body>Score Display</body></html>{{end}}
`))
	mockDB := &testutil.MockDB{}
	handler := New(mockDB, templates)
	return handler, mockDB
}

func TestLimitMasterScoresKeepsMostRecentEntries(t *testing.T) {
	var scores []models.MasterScore
	for i := 1; i <= 12; i++ {
		scores = append(scores, models.MasterScore{
			Date:  "2024-01-" + strconv.Itoa(i),
			Score: float64(i),
		})
	}

	limited := limitMasterScores(scores, historyPreviewLimit)

	if len(limited) != historyPreviewLimit {
		t.Fatalf("expected %d scores, got %d", historyPreviewLimit, len(limited))
	}
	if limited[0].Date != "2024-01-3" {
		t.Fatalf("expected first retained score to be 2024-01-3, got %s", limited[0].Date)
	}
	if limited[len(limited)-1].Date != "2024-01-12" {
		t.Fatalf("expected latest retained score to be 2024-01-12, got %s", limited[len(limited)-1].Date)
	}
}

func TestHandleHome(t *testing.T) {
	handler, mockDB := setupTestHandler()

	// Mock the database calls
	mockDB.GetUserProfileFunc = func() (*models.UserProfile, error) {
		return &models.UserProfile{
			BirthDate: "1990-01-01",
			Sex:       "male",
			HeightCm:  180.0,
		}, nil
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.HandleHome(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
}

func TestHandleSettings(t *testing.T) {
	handler, mockDB := setupTestHandler()

	// Mock the database calls
	mockDB.GetUserProfileFunc = func() (*models.UserProfile, error) {
		return &models.UserProfile{
			BirthDate: "1990-01-01",
			Sex:       "male",
			HeightCm:  180.0,
		}, nil
	}

	req, err := http.NewRequest("GET", "/settings", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.HandleSettings(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
}

func TestHandleRationale(t *testing.T) {
	handler, _ := setupTestHandler()

	req, err := http.NewRequest("GET", "/rationale", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.HandleRationale(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
}

func TestHandleAddHealthMetrics(t *testing.T) {
	handler, mockDB := setupTestHandler()

	// Mock the save function
	mockDB.SaveHealthMetricsFunc = func(m models.HealthMetrics) error {
		if m.SleepScore != 80 {
			t.Errorf("Expected SleepScore 80, got %d", m.SleepScore)
		}
		if m.SystolicBP != 118 || m.DiastolicBP != 76 {
			t.Errorf("Expected BP 118/76, got %d/%d", m.SystolicBP, m.DiastolicBP)
		}
		return nil
	}

	formData := "sleep_score=80&waist_cm=85.0&body_weight_kg=75.0&rhr=60&systolic_bp=118&diastolic_bp=76&nutrition_score=7.5"
	req, err := http.NewRequest("POST", "/add-health-metrics", strings.NewReader(formData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler.HandleAddHealthMetrics(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("Expected status code %d, got %d", http.StatusNoContent, status)
	}

	// Check if the success trigger is set
	trigger := rr.Header().Get("HX-Trigger")
	if trigger == "" {
		t.Error("Expected HX-Trigger header to be set")
	}
}

func TestHandleHealthMetricsUsesHistoryPreviewLimit(t *testing.T) {
	handler, mockDB := setupTestHandler()

	mockDB.GetRecentHealthMetricsFunc = func(limit int) ([]models.HealthMetrics, error) {
		if limit != historyPreviewLimit {
			t.Fatalf("expected limit %d, got %d", historyPreviewLimit, limit)
		}
		return []models.HealthMetrics{}, nil
	}

	req, err := http.NewRequest("GET", "/health-metrics", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.HandleHealthMetrics(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
}

func TestHandleAddHealthMetricsInvalidMethod(t *testing.T) {
	handler, _ := setupTestHandler()

	req, err := http.NewRequest("GET", "/add-health-metrics", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.HandleAddHealthMetrics(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code %d, got %d", http.StatusMethodNotAllowed, status)
	}
}

func TestHandleAddFitnessMetrics(t *testing.T) {
	handler, mockDB := setupTestHandler()

	// Mock the save function
	mockDB.SaveFitnessMetricsFunc = func(m models.FitnessMetrics) error {
		if m.VO2Max != 45.0 {
			t.Errorf("Expected VO2Max 45.0, got %f", m.VO2Max)
		}
		if m.LowerBodyWeight != 180.0 || m.LowerBodyReps != 12 {
			t.Errorf("Expected leg press 180.0 x 12, got %.1f x %d", m.LowerBodyWeight, m.LowerBodyReps)
		}
		return nil
	}

	formData := "vo2_max=45.0&workouts=4&daily_steps=10000&mobility=3&cardio_recovery=25&leg_press_set=180x12&dead_hang_seconds=75"
	req, err := http.NewRequest("POST", "/add-fitness-metrics", strings.NewReader(formData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler.HandleAddFitnessMetrics(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("Expected status code %d, got %d", http.StatusNoContent, status)
	}

	// Check if the success trigger is set
	trigger := rr.Header().Get("HX-Trigger")
	if trigger == "" {
		t.Error("Expected HX-Trigger header to be set")
	}
}

func TestHandleFitnessMetricsUsesHistoryPreviewLimit(t *testing.T) {
	handler, mockDB := setupTestHandler()

	mockDB.GetRecentFitnessMetricsFunc = func(limit int) ([]models.FitnessMetrics, error) {
		if limit != historyPreviewLimit {
			t.Fatalf("expected limit %d, got %d", historyPreviewLimit, limit)
		}
		return []models.FitnessMetrics{}, nil
	}

	req, err := http.NewRequest("GET", "/fitness-metrics", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.HandleFitnessMetrics(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
}

func TestParseWeightAndReps(t *testing.T) {
	weight, reps, err := parseWeightAndReps("180x12")
	if err != nil {
		t.Fatalf("Expected valid parse, got error: %v", err)
	}
	if weight != 180 || reps != 12 {
		t.Fatalf("Expected 180 x 12, got %.1f x %d", weight, reps)
	}

	if _, _, err := parseWeightAndReps("bad-input"); err == nil {
		t.Fatal("Expected invalid format to return an error")
	}
}

func TestHandleAddCognitionMetrics(t *testing.T) {
	handler, mockDB := setupTestHandler()

	// Mock the save function
	mockDB.SaveCognitionMetricsFunc = func(m models.CognitionMetrics) error {
		if m.Mindfulness != 4 {
			t.Errorf("Expected Mindfulness 4, got %d", m.Mindfulness)
		}
		if m.StressScore != 2 || m.SocialDays != 5 {
			t.Errorf("Expected stress/social 2/5, got %d/%d", m.StressScore, m.SocialDays)
		}
		return nil
	}

	formData := "mindfulness=4&deep_learning=60&stress_score=2&social_days=5"
	req, err := http.NewRequest("POST", "/add-cognition-metrics", strings.NewReader(formData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler.HandleAddCognitionMetrics(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("Expected status code %d, got %d", http.StatusNoContent, status)
	}

	// Check if the success trigger is set
	trigger := rr.Header().Get("HX-Trigger")
	if trigger == "" {
		t.Error("Expected HX-Trigger header to be set")
	}
}

func TestHandleCognitionMetricsUsesHistoryPreviewLimit(t *testing.T) {
	handler, mockDB := setupTestHandler()

	mockDB.GetRecentCognitionMetricsFunc = func(limit int) ([]models.CognitionMetrics, error) {
		if limit != historyPreviewLimit {
			t.Fatalf("expected limit %d, got %d", historyPreviewLimit, limit)
		}
		return []models.CognitionMetrics{}, nil
	}

	req, err := http.NewRequest("GET", "/cognition-metrics", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.HandleCognitionMetrics(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
}

func TestHandleDeleteHealthMetric(t *testing.T) {
	handler, mockDB := setupTestHandler()

	// Mock the delete function
	mockDB.DeleteHealthMetricsFunc = func(date string) error {
		if date != "2023-01-01" {
			t.Errorf("Expected date '2023-01-01', got '%s'", date)
		}
		return nil
	}

	req, err := http.NewRequest("DELETE", "/delete-health-metric?date=2023-01-01", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.HandleDeleteHealthMetric(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check if the success trigger is set
	trigger := rr.Header().Get("HX-Trigger")
	if trigger == "" {
		t.Error("Expected HX-Trigger header to be set")
	}
}

func TestHandleDeleteFitnessMetric(t *testing.T) {
	handler, mockDB := setupTestHandler()

	// Mock the delete function
	mockDB.DeleteFitnessMetricsFunc = func(date string) error {
		if date != "2023-01-01" {
			t.Errorf("Expected date '2023-01-01', got '%s'", date)
		}
		return nil
	}

	req, err := http.NewRequest("DELETE", "/delete-fitness-metric?date=2023-01-01", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.HandleDeleteFitnessMetric(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check if the success trigger is set
	trigger := rr.Header().Get("HX-Trigger")
	if trigger == "" {
		t.Error("Expected HX-Trigger header to be set")
	}
}

func TestHandleDeleteCognitionMetric(t *testing.T) {
	handler, mockDB := setupTestHandler()

	// Mock the delete function
	mockDB.DeleteCognitionMetricsFunc = func(date string) error {
		if date != "2023-01-01" {
			t.Errorf("Expected date '2023-01-01', got '%s'", date)
		}
		return nil
	}

	req, err := http.NewRequest("DELETE", "/delete-cognition-metric?date=2023-01-01", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.HandleDeleteCognitionMetric(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check if the success trigger is set
	trigger := rr.Header().Get("HX-Trigger")
	if trigger == "" {
		t.Error("Expected HX-Trigger header to be set")
	}
}

func TestHandleUpdateProfile(t *testing.T) {
	handler, mockDB := setupTestHandler()

	// Mock the save function
	mockDB.SaveUserProfileFunc = func(profile models.UserProfile) error {
		if profile.BirthDate != "1990-01-01" {
			t.Errorf("Expected BirthDate '1990-01-01', got '%s'", profile.BirthDate)
		}
		return nil
	}

	formData := "birth_date=1990-01-01&sex=male&height_cm=180.0"
	req, err := http.NewRequest("POST", "/update-profile", strings.NewReader(formData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler.HandleUpdateProfile(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check if the success trigger is set
	trigger := rr.Header().Get("HX-Trigger")
	if trigger == "" {
		t.Error("Expected HX-Trigger header to be set")
	}
}

func TestHandleSubscribe(t *testing.T) {
	handler, mockDB := setupTestHandler()

	// Mock the save function
	mockDB.SavePushSubscriptionFunc = func(sub models.PushSubscription) error {
		if sub.Endpoint != "https://example.com/subscription" {
			t.Errorf("Expected Endpoint 'https://example.com/subscription', got '%s'", sub.Endpoint)
		}
		return nil
	}

	subReq := models.PushSubscriptionRequest{
		Subscription: models.PushSubscription{
			Endpoint: "https://example.com/subscription",
			P256dh:   "p256dh_key",
			Auth:     "auth_key",
		},
		ReminderDay:  1,
		ReminderTime: "09:00",
		Timezone:     "UTC",
	}

	jsonData, _ := json.Marshal(subReq)
	req, err := http.NewRequest("POST", "/subscribe", bytes.NewReader(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleSubscribe(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check if the response is JSON
	responseBody, _ := io.ReadAll(rr.Body)
	var response map[string]string
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		t.Errorf("Expected JSON response, got error: %v", err)
	}
	if response["status"] != "success" {
		t.Errorf("Expected status 'success', got '%s'", response["status"])
	}
}

func TestHandleAppHealth(t *testing.T) {
	handler, _ := setupTestHandler()

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.HandleAppHealth(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	expected := "OK"
	if rr.Body.String() != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, rr.Body.String())
	}
}

func TestParseFormInt(t *testing.T) {
	req, err := http.NewRequest("POST", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Add form values
	req.Form = make(map[string][]string)
	req.Form["test_field"] = []string{"123"}

	result, err := parseFormInt(req, "test_field")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 123 {
		t.Errorf("Expected 123, got %d", result)
	}
}

func TestParseFormFloat(t *testing.T) {
	req, err := http.NewRequest("POST", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Add form values
	req.Form = make(map[string][]string)
	req.Form["test_field"] = []string{"12.34"}

	result, err := parseFormFloat(req, "test_field")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 12.34 {
		t.Errorf("Expected 12.34, got %f", result)
	}
}
