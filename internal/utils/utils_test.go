package utils

import (
	"health-balance/internal/models"
	"testing"
	"time"
)

func TestGetCurrentWeekSundayDate(t *testing.T) {
	dateStr := GetCurrentWeekSundayDate()

	parsed, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		t.Errorf("GetCurrentWeekSundayDate() returned invalid format: %v", err)
	}

	if parsed.Weekday() != time.Sunday {
		t.Errorf("GetCurrentWeekSundayDate() returned %v, which is a %v, not a Sunday", dateStr, parsed.Weekday())
	}

	now := time.Now()
	// The returned Sunday should be >= today (either today if Sunday, or upcoming)
	if parsed.Before(now.Truncate(24 * time.Hour)) {
		t.Errorf("GetCurrentWeekSundayDate() returned a past Sunday %v, should return current or upcoming", dateStr)
	}

	// Should be within 7 days of today
	daysUntilSunday := parsed.Sub(now.Truncate(24*time.Hour)).Hours() / 24
	if daysUntilSunday > 7 {
		t.Errorf("GetCurrentWeekSundayDate() returned Sunday %v which is more than 7 days away", dateStr)
	}
}

func TestGetAge(t *testing.T) {
	refTime := time.Date(2025, 12, 27, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		birthDate string
		wantAge   int
		wantErr   bool
	}{
		{
			name:      "Birthday was yesterday",
			birthDate: "1990-12-26",
			wantAge:   35,
			wantErr:   false,
		},
		{
			name:      "Birthday is today",
			birthDate: "1990-12-27",
			wantAge:   35,
			wantErr:   false,
		},
		{
			name:      "Birthday is tomorrow (not occurred yet)",
			birthDate: "1990-12-28",
			wantAge:   34,
			wantErr:   false,
		},
		{
			name:      "Birthday is months away",
			birthDate: "1990-06-01",
			wantAge:   35,
			wantErr:   false,
		},
		{
			name:      "Empty birth date",
			birthDate: "",
			wantAge:   0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &models.UserProfile{BirthDate: tt.birthDate}
			got, err := GetAge(p, refTime)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantAge {
				t.Errorf("GetAge() = %v, want %v", got, tt.wantAge)
			}
		})
	}
}
