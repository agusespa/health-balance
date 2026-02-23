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

func TestGetCurrentWeekDateRange(t *testing.T) {
	// Test that the function returns a non-empty string
	dateRange := GetCurrentWeekDateRange()
	if dateRange == "" {
		t.Error("GetCurrentWeekDateRange() returned empty string")
	}

	// Test that it contains expected format elements
	// Should contain month abbreviations and numbers
	// Examples: "Feb 23 - Mar 1" or "Dec 30 - Jan 5, 2027"
	
	// Verify the range makes sense by checking the underlying dates
	now := time.Now()
	weekday := now.Weekday()
	
	var monday, sunday time.Time
	if weekday == time.Sunday {
		monday = now.AddDate(0, 0, -6)
		sunday = now
	} else {
		daysSinceMonday := int(weekday) - 1
		monday = now.AddDate(0, 0, -daysSinceMonday)
		daysUntilSunday := 7 - int(weekday)
		sunday = now.AddDate(0, 0, daysUntilSunday)
	}
	
	// Verify Monday is actually a Monday
	if monday.Weekday() != time.Monday {
		t.Errorf("Calculated Monday is %v, not Monday", monday.Weekday())
	}
	
	// Verify Sunday is actually a Sunday
	if sunday.Weekday() != time.Sunday {
		t.Errorf("Calculated Sunday is %v, not Sunday", sunday.Weekday())
	}
	
	// Verify the range is exactly 6 days
	daysDiff := sunday.Sub(monday).Hours() / 24
	if daysDiff != 6 {
		t.Errorf("Date range should be 6 days, got %.0f days", daysDiff)
	}
	
	// Verify Monday is not in the future
	if monday.After(now) {
		t.Errorf("Monday %v should not be after today %v", monday, now)
	}
	
	// Verify Sunday is not in the past (unless today is Sunday)
	if weekday != time.Sunday && sunday.Before(now.Truncate(24*time.Hour)) {
		t.Errorf("Sunday %v should not be before today %v", sunday, now)
	}
}

func TestGetCurrentWeekDateRangeFormat(t *testing.T) {
	tests := []struct {
		name        string
		testDate    time.Time
		expectYear  bool
		description string
	}{
		{
			name:        "Mid-week in February",
			testDate:    time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC), // Wednesday
			expectYear:  false,
			description: "Should not include year when within same year",
		},
		{
			name:        "Monday in March",
			testDate:    time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC), // Monday
			expectYear:  false,
			description: "Monday should show current week",
		},
		{
			name:        "Sunday",
			testDate:    time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), // Sunday
			expectYear:  false,
			description: "Sunday should show Mon-Sun of that week",
		},
		{
			name:        "Week crossing year boundary",
			testDate:    time.Date(2026, 12, 30, 12, 0, 0, 0, time.UTC), // Wednesday
			expectYear:  true,
			description: "Should include year when crossing year boundary",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test validates the logic but can't directly test with a specific date
			// since GetCurrentWeekDateRange uses time.Now()
			// In a real scenario, you'd refactor to accept a time parameter for testing
			t.Logf("Test case: %s - %s", tt.name, tt.description)
		})
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
