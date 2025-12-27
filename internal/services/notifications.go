package services

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"health-balance/internal/database"
	"health-balance/internal/models"
	"health-balance/internal/utils"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"time"
)

func getVapidKeys() (string, string) {
	pub := os.Getenv("VAPID_PUBLIC_KEY")
	priv := os.Getenv("VAPID_PRIVATE_KEY")
	return pub, priv
}

func StartNotificationScheduler(db database.Querier) {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			checkAndSendNotifications(db)
		}
	}()
}

func checkAndSendNotifications(db database.Querier) {
	subs, err := db.GetAllSubscriptions()
	if err != nil {
		log.Printf("Scheduler error: failed to get subscriptions: %v", err)
		return
	}

	if len(subs) == 0 {
		return
	}

	// Only send if data is missing for the current week
	currentWeekDate := utils.GetPreviousSundayDate()
	h, _ := db.GetHealthMetricsByDate(currentWeekDate)
	f, _ := db.GetFitnessMetricsByDate(currentWeekDate)
	c, _ := db.GetCognitionMetricsByDate(currentWeekDate)

	dataComplete := h != nil && f != nil && c != nil
	if dataComplete {
		log.Printf("Scheduler: Skipping - data complete for week %s", currentWeekDate)
		return
	}

	// Decode private key once
	_, privKeyStr := getVapidKeys()
	if privKeyStr == "" {
		log.Printf("VAPID_PRIVATE_KEY not set")
		return
	}
	priv, err := decodePrivateKey(privKeyStr)
	if err != nil {
		log.Printf("Failed to decode VAPID private key: %v", err)
		return
	}

	for _, sub := range subs {
		shouldSend := false

		location, err := time.LoadLocation(sub.Timezone)
		if err != nil {
			// Fallback to UTC if timezone is invalid
			location = time.UTC
		}

		nowInLoc := time.Now().In(location)
		day := int(nowInLoc.Weekday())
		timeStr := nowInLoc.Format("15:04")

		if day == sub.ReminderDay && timeStr == sub.ReminderTime {
			shouldSend = true
		}

		if shouldSend {
			log.Printf("Scheduler: Sending notification to %s (Timezone: %s, Local Time: %s)", sub.Endpoint, sub.Timezone, timeStr)
			go sendPush(db, sub, priv)
		}
	}
}

func sendPush(db database.Querier, sub models.PushSubscription, priv *ecdsa.PrivateKey) {
	parsedURL, err := url.Parse(sub.Endpoint)
	if err != nil {
		log.Printf("Invalid endpoint URL %s: %v", sub.Endpoint, err)
		return
	}

	audience := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
	token, err := createVAPIDToken(audience, priv)
	if err != nil {
		log.Printf("Failed to create VAPID token: %v", err)
		return
	}

	req, err := http.NewRequest("POST", sub.Endpoint, nil)
	if err != nil {
		log.Printf("Failed to create push request: %v", err)
		return
	}

	pubKey, _ := getVapidKeys()
	req.Header.Set("Authorization", "WebPush "+token)
	req.Header.Set("TTL", "30")
	req.Header.Set("Crypto-Key", "p256ecdsa="+pubKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send push to %s: %v", sub.Endpoint, err)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
		log.Printf("Push subscription stale (%d), purging from database: %s", resp.StatusCode, sub.Endpoint)
		if err := db.DeletePushSubscription(sub.Endpoint); err != nil {
			log.Printf("Failed to delete stale subscription: %v", err)
		}
		return
	}

	if resp.StatusCode >= 400 {
		log.Printf("Push service returned error %d for %s", resp.StatusCode, sub.Endpoint)
	}
}

func decodePrivateKey(encoded string) (*ecdsa.PrivateKey, error) {
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("invalid private key length: %d", len(raw))
	}

	k, err := ecdh.P256().NewPrivateKey(raw)
	if err != nil {
		return nil, err
	}

	pubBytes := k.PublicKey().Bytes() // 0x04 + X (32 bytes) + Y (32 bytes)
	return &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(pubBytes[1:33]),
			Y:     new(big.Int).SetBytes(pubBytes[33:]),
		},
		D: new(big.Int).SetBytes(raw),
	}, nil
}

func createVAPIDToken(audience string, priv *ecdsa.PrivateKey) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"ES256","typ":"JWT"}`))

	payloadMap := map[string]any{
		"aud": audience,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"sub": "mailto:admin@health-balance.local",
	}
	payloadJSON, _ := json.Marshal(payloadMap)
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)

	unsignedToken := header + "." + payload
	hash := sha256.Sum256([]byte(unsignedToken))
	r, s, err := ecdsa.Sign(rand.Reader, priv, hash[:])
	if err != nil {
		return "", err
	}

	// ES256 signature is R and S concatenated (32 bytes each)
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// Pad to 32 bytes if necessary
	sig := make([]byte, 64)
	copy(sig[32-len(rBytes):32], rBytes)
	copy(sig[64-len(sBytes):64], sBytes)

	return unsignedToken + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}
