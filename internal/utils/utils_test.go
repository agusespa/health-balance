package utils

import (
	"health-balance/internal/models"
	"testing"
	"time"
)

func TestGetActiveWeekEndDate(t *testing.T) {
	dateStr := GetActiveWeekEndDate()

	parsed, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		t.Fatalf("GetActiveWeekEndDate() returned invalid format: %v", err)
	}

	if parsed.Weekday() != time.Friday {
		t.Fatalf("GetActiveWeekEndDate() returned %v, which is a %v, not a Friday", dateStr, parsed.Weekday())
	}

	now := time.Now()
	currentDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	daysSinceSaturday := (int(currentDate.Weekday()) - int(time.Saturday) + 7) % 7
	expected := currentDate.AddDate(0, 0, -daysSinceSaturday-1)
	if parsed.Format("2006-01-02") != expected.Format("2006-01-02") {
		t.Fatalf("GetActiveWeekEndDate() = %s, want %s", dateStr, expected.Format("2006-01-02"))
	}
}

func TestGetActiveWeekDateRange(t *testing.T) {
	dateRange := GetActiveWeekDateRange()
	if dateRange == "" {
		t.Fatal("GetActiveWeekDateRange() returned empty string")
	}

	now := time.Now()
	weekRange := getActiveWeekRange(now)

	if weekRange.start.Weekday() != time.Saturday {
		t.Fatalf("Calculated range start is %v, not Saturday", weekRange.start.Weekday())
	}
	if weekRange.end.Weekday() != time.Friday {
		t.Fatalf("Calculated range end is %v, not Friday", weekRange.end.Weekday())
	}

	daysDiff := weekRange.end.Sub(weekRange.start).Hours() / 24
	if daysDiff != 6 {
		t.Fatalf("Date range should be 6 days, got %.0f days", daysDiff)
	}

	expected := formatWeekRange(weekRange.start, weekRange.end)
	if dateRange != expected {
		t.Fatalf("GetActiveWeekDateRange() = %q, want %q", dateRange, expected)
	}
}

func TestGetActiveWeekRange(t *testing.T) {
	tests := []struct {
		name      string
		now       time.Time
		wantStart string
		wantEnd   string
	}{
		{
			name:      "Saturday starts new entry window",
			now:       time.Date(2026, 4, 4, 9, 0, 0, 0, time.UTC),
			wantStart: "2026-03-28",
			wantEnd:   "2026-04-03",
		},
		{
			name:      "Friday still edits previous completed week",
			now:       time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			wantStart: "2026-03-21",
			wantEnd:   "2026-03-27",
		},
		{
			name:      "Tuesday stays in same entry window",
			now:       time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC),
			wantStart: "2026-03-21",
			wantEnd:   "2026-03-27",
		},
		{
			name:      "Cross year boundary",
			now:       time.Date(2027, 1, 2, 8, 0, 0, 0, time.UTC),
			wantStart: "2026-12-26",
			wantEnd:   "2027-01-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getActiveWeekRange(tt.now)
			if got.start.Format("2006-01-02") != tt.wantStart {
				t.Fatalf("start = %s, want %s", got.start.Format("2006-01-02"), tt.wantStart)
			}
			if got.end.Format("2006-01-02") != tt.wantEnd {
				t.Fatalf("end = %s, want %s", got.end.Format("2006-01-02"), tt.wantEnd)
			}
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
