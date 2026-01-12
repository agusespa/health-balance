package services

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"os"
	"strings"
	"testing"

	"health-balance/internal/models"
	"health-balance/internal/testutil"
)

func TestGetVapidKeys(t *testing.T) {
	_ = os.Setenv("VAPID_PUBLIC_KEY", "test_public_key")
	_ = os.Setenv("VAPID_PRIVATE_KEY", "test_private_key")
	defer func() {
		_ = os.Unsetenv("VAPID_PUBLIC_KEY")
		_ = os.Unsetenv("VAPID_PRIVATE_KEY")
	}()

	public, private := getVapidKeys()
	if public != "test_public_key" {
		t.Errorf("Expected public key 'test_public_key', got '%s'", public)
	}
	if private != "test_private_key" {
		t.Errorf("Expected private key 'test_private_key', got '%s'", private)
	}
}

func TestGetVapidKeysMissing(t *testing.T) {
	_ = os.Unsetenv("VAPID_PUBLIC_KEY")
	_ = os.Unsetenv("VAPID_PRIVATE_KEY")

	public, private := getVapidKeys()
	if public != "" {
		t.Errorf("Expected empty public key, got '%s'", public)
	}
	if private != "" {
		t.Errorf("Expected empty private key, got '%s'", private)
	}
}

func TestCheckAndSendNotifications(t *testing.T) {
	mockDB := &testutil.MockDB{
		GetAllSubscriptionsFunc: func() ([]models.PushSubscription, error) {
			return []models.PushSubscription{}, nil
		},
	}

	// Call checkAndSendNotifications - should return early since no subscriptions
	checkAndSendNotifications(mockDB)
}

func TestDecodePrivateKey(t *testing.T) {
	_, err := decodePrivateKey("invalid_base64")
	if err == nil {
		t.Error("Expected error when decoding invalid base64, but got none")
	}

	shortKey := "AQIDBAUGBwg="
	_, err = decodePrivateKey(shortKey)
	if err == nil {
		t.Error("Expected error when decoding short key, but got none")
	}
}

func TestCreateVAPIDToken(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	token, err := createVAPIDToken("https://example.com", privateKey)
	if err != nil {
		t.Errorf("Unexpected error when creating VAPID token: %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty VAPID token")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("Expected 3 parts in JWT token, got %d", len(parts))
	}
}

func TestCheckAndSendNotificationsWithDataComplete(t *testing.T) {
	_ = os.Setenv("VAPID_PRIVATE_KEY", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	defer func() {
		_ = os.Unsetenv("VAPID_PRIVATE_KEY")
	}()

	mockDB := &testutil.MockDB{
		GetAllSubscriptionsFunc: func() ([]models.PushSubscription, error) {
			return []models.PushSubscription{
				{
					Endpoint:     "https://example.com/push",
					P256dh:       "p256dh_key",
					Auth:         "auth_key",
					ReminderDay:  1, // Monday
					ReminderTime: "09:00",
					Timezone:     "UTC",
				},
			}, nil
		},
		GetHealthMetricsByDateFunc: func(date string) (*models.HealthMetrics, error) {
			return &models.HealthMetrics{}, nil
		},
		GetFitnessMetricsByDateFunc: func(date string) (*models.FitnessMetrics, error) {
			return &models.FitnessMetrics{}, nil
		},
		GetCognitionMetricsByDateFunc: func(date string) (*models.CognitionMetrics, error) {
			return &models.CognitionMetrics{}, nil
		},
	}

	// Call checkAndSendNotifications - should skip sending since data is complete
	checkAndSendNotifications(mockDB)
}

func TestCheckAndSendNotificationsMissingData(t *testing.T) {
	_ = os.Setenv("VAPID_PRIVATE_KEY", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	defer func() {
		_ = os.Unsetenv("VAPID_PRIVATE_KEY")
	}()

	mockDB := &testutil.MockDB{
		GetAllSubscriptionsFunc: func() ([]models.PushSubscription, error) {
			return []models.PushSubscription{
				{
					Endpoint:     "https://example.com/push",
					P256dh:       "p256dh_key",
					Auth:         "auth_key",
					ReminderDay:  1, // Monday
					ReminderTime: "09:00",
					Timezone:     "UTC",
				},
			}, nil
		},
		GetHealthMetricsByDateFunc: func(date string) (*models.HealthMetrics, error) {
			return nil, nil
		},
		GetFitnessMetricsByDateFunc: func(date string) (*models.FitnessMetrics, error) {
			return nil, nil
		},
		GetCognitionMetricsByDateFunc: func(date string) (*models.CognitionMetrics, error) {
			return nil, nil
		},
	}

	// Call checkAndSendNotifications - should attempt to send notification since data is missing
	checkAndSendNotifications(mockDB)
}
