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
	tests := []struct {
		name      string
		birthDate string
		wantAge   int
		wantErr   bool
	}{
		{
			name:      "Birthday already passed this year",
			birthDate: "1990-01-01",
			wantAge:   35, // 2025 - 1990
			wantErr:   false,
		},
		{
			name:      "Birthday is tomorrow (hasn't occurred yet)",
			birthDate: "1990-12-27",
			wantAge:   34, // Should be 35 - 1
			wantErr:   false,
		},
		{
			name:      "Birth date is today",
			birthDate: "1990-12-26",
			wantAge:   35,
			wantErr:   false,
		},
		{
			name:      "Empty birth date",
			birthDate: "",
			wantAge:   0,
			wantErr:   true,
		},
		{
			name:      "Invalid date format",
			birthDate: "01/01/1990",
			wantAge:   0,
			wantErr:   true,
		},
		{
			name:      "Non-existent date",
			birthDate: "1990-02-30",
			wantAge:   0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &models.UserProfile{BirthDate: tt.birthDate}
			got, err := GetAge(p)

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
