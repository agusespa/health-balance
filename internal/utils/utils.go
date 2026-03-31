package utils

import (
	"errors"
	"fmt"
	"health-balance/internal/models"
	"time"
)

// GetActiveWeekEndDate returns the Friday date for the most recently completed
// Saturday-Friday reporting week. That week is the one users can edit during
// the current Saturday-Friday entry window.
func GetActiveWeekEndDate() string {
	return getActiveWeekRange(time.Now()).end.Format("2006-01-02")
}

// GetActiveWeekDateRange returns the Saturday-Friday range for the week that is
// currently editable on the dashboard.
func GetActiveWeekDateRange() string {
	weekRange := getActiveWeekRange(time.Now())
	return formatWeekRange(weekRange.start, weekRange.end)
}

type weekRange struct {
	start time.Time
	end   time.Time
}

func getActiveWeekRange(now time.Time) weekRange {
	currentDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	daysSinceSaturday := (int(currentDate.Weekday()) - int(time.Saturday) + 7) % 7
	windowStart := currentDate.AddDate(0, 0, -daysSinceSaturday)
	end := windowStart.AddDate(0, 0, -1)
	start := end.AddDate(0, 0, -6)
	return weekRange{start: start, end: end}
}

func formatWeekRange(start, end time.Time) string {
	// Format: "Feb 23 - Mar 1" or "Dec 30 - Jan 5, 2027" if crossing year
	if start.Year() == end.Year() {
		return fmt.Sprintf("%s %d - %s %d",
			start.Month().String()[:3], start.Day(),
			end.Month().String()[:3], end.Day())
	}
	return fmt.Sprintf("%s %d - %s %d, %d",
		start.Month().String()[:3], start.Day(),
		end.Month().String()[:3], end.Day(), end.Year())
}

func GetAge(p *models.UserProfile, now time.Time) (int, error) {
	if p.BirthDate == "" {
		return 0, errors.New("birth date is missing from user profile")
	}

	birthDate, err := time.Parse("2006-01-02", p.BirthDate)
	if err != nil {
		return 0, fmt.Errorf("invalid birth date format '%s': %w", p.BirthDate, err)
	}

	age := now.Year() - birthDate.Year()

	if now.Month() < birthDate.Month() ||
		(now.Month() == birthDate.Month() && now.Day() < birthDate.Day()) {
		age--
	}

	return age, nil
}
