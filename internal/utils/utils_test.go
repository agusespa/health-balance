package utils

import (
	"health-balance/internal/models"
	"testing"
	"time"
)

func TestGetPreviousSundayDate(t *testing.T) {
	dateStr := GetPreviousSundayDate()

	parsed, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		t.Errorf("GetPreviousSundayDate() returned invalid format: %v", err)
	}

	if parsed.Weekday() != time.Sunday {
		t.Errorf("GetPreviousSundayDate() returned %v, which is a %v, not a Sunday", dateStr, parsed.Weekday())
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
