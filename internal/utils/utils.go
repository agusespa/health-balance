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
