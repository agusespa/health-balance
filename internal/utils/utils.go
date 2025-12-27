package utils

import (
	"errors"
	"fmt"
	"health-balance/internal/models"
	"time"
)

func GetPreviousSundayDate() string {
	now := time.Now()
	weekday := now.Weekday()
	daysSinceSunday := int(weekday)
	previousSunday := now.AddDate(0, 0, -daysSinceSunday)
	return previousSunday.Format("2006-01-02")
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
