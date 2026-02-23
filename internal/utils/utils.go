package utils

import (
	"errors"
	"fmt"
	"health-balance/internal/models"
	"time"
)

// GetCurrentWeekSundayDate returns the Sunday date for the current week
// If today is Sunday, it returns today's date
// If today is Monday-Saturday, it returns the upcoming Sunday's date
func GetCurrentWeekSundayDate() string {
	now := time.Now()
	weekday := now.Weekday()

	if weekday == time.Sunday {
		// If today is Sunday, this is the current week
		return now.Format("2006-01-02")
	}

	// For Monday-Saturday, the current week is the upcoming Sunday
	daysUntilSunday := 7 - int(weekday)
	nextSunday := now.AddDate(0, 0, daysUntilSunday)
	return nextSunday.Format("2006-01-02")
}

// GetCurrentWeekDateRange returns the Monday and Sunday dates for the current week
// Returns in format "Mon DD - Mon DD" or "Mon DD - Mon DD, YYYY" if crossing year boundary
func GetCurrentWeekDateRange() string {
	now := time.Now()
	weekday := now.Weekday()

	var monday, sunday time.Time

	if weekday == time.Sunday {
		// If today is Sunday, go back 6 days to get Monday
		monday = now.AddDate(0, 0, -6)
		sunday = now
	} else {
		// For Monday-Saturday, calculate the Monday of this week and upcoming Sunday
		daysSinceMonday := int(weekday) - 1
		monday = now.AddDate(0, 0, -daysSinceMonday)
		daysUntilSunday := 7 - int(weekday)
		sunday = now.AddDate(0, 0, daysUntilSunday)
	}

	// Format: "Feb 23 - Mar 1" or "Dec 30 - Jan 5, 2027" if crossing year
	if monday.Year() == sunday.Year() {
		return fmt.Sprintf("%s %d - %s %d",
			monday.Month().String()[:3], monday.Day(),
			sunday.Month().String()[:3], sunday.Day())
	}
	return fmt.Sprintf("%s %d - %s %d, %d",
		monday.Month().String()[:3], monday.Day(),
		sunday.Month().String()[:3], sunday.Day(), sunday.Year())
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
