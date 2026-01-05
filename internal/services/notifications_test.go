package services

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"health-balance/internal/models"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Mock DB for testing
type mockDB struct{}

// Implement all Querier interface methods with dummy data or empty implementations
func (m *mockDB) GetAllDatesWithData() ([]string, error)                             { return nil, nil }
func (m *mockDB) GetRecentHealthMetrics(limit int) ([]models.HealthMetrics, error)   { return nil, nil }
func (m *mockDB) GetRecentFitnessMetrics(limit int) ([]models.FitnessMetrics, error) { return nil, nil }
func (m *mockDB) GetRecentCognitionMetrics(limit int) ([]models.CognitionMetrics, error) {
	return nil, nil
}
func (m *mockDB) SaveHealthMetrics(m1 models.HealthMetrics) error                   { return nil }
func (m *mockDB) SaveFitnessMetrics(m1 models.FitnessMetrics) error                 { return nil }
func (m *mockDB) SaveCognitionMetrics(m1 models.CognitionMetrics) error             { return nil }
func (m *mockDB) GetHealthMetricsByDate(date string) (*models.HealthMetrics, error) { return nil, nil }
func (m *mockDB) GetFitnessMetricsByDate(date string) (*models.FitnessMetrics, error) {
	return nil, nil
}
func (m *mockDB) GetCognitionMetricsByDate(date string) (*models.CognitionMetrics, error) {
	return nil, nil
}
func (m *mockDB) DeleteHealthMetrics(date string) error                     { return nil }
func (m *mockDB) DeleteFitnessMetrics(date string) error                    { return nil }
func (m *mockDB) DeleteCognitionMetrics(date string) error                  { return nil }
func (m *mockDB) GetRHRBaseline() (int, error)                              { return 0, nil }
func (m *mockDB) GetUserProfile() (*models.UserProfile, error)              { return nil, nil }
func (m *mockDB) SaveUserProfile(profile models.UserProfile) error          { return nil }
func (m *mockDB) SavePushSubscription(sub models.PushSubscription) error    { return nil }
func (m *mockDB) GetAllSubscriptions() ([]models.PushSubscription, error)   { return nil, nil }
func (m *mockDB) GetAnyPushSubscription() (*models.PushSubscription, error) { return nil, nil }
func (m *mockDB) DeletePushSubscription(endpoint string) error              { return nil }
func (m *mockDB) Close() error                                              { return nil }

func TestVapidHeaders(t *testing.T) {
	// Generate temporary VAPID keys for testing
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	ecdhPriv, err := privKey.ECDH()
	if err != nil {
		t.Fatalf("Failed to convert private key to ECDH: %v", err)
	}
	pubKeyBytes := ecdhPriv.PublicKey().Bytes()
	pubKeyStr := base64.RawURLEncoding.EncodeToString(pubKeyBytes)

	// Set env var for public key (needed by sendPush)
	os.Setenv("VAPID_PUBLIC_KEY", pubKeyStr)
	defer os.Unsetenv("VAPID_PUBLIC_KEY")

	// Create a test server to capture the request headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "vapid t=") {
			t.Errorf("Expected Authorization header to start with 'vapid t=', got '%s'", authHeader)
		}
		if !strings.Contains(authHeader, ", k=") {
			t.Errorf("Expected Authorization header to contain ', k=', got '%s'", authHeader)
		}

		// Check that Crypto-Key header is NOT present
		if r.Header.Get("Crypto-Key") != "" {
			t.Errorf("Expected Crypto-Key header to be empty, got '%s'", r.Header.Get("Crypto-Key"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sub := models.PushSubscription{
		Endpoint: server.URL,
	}

	sendPush(&mockDB{}, sub, privKey)
}
