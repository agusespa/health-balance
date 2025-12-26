package models

// GetVO2MaxBaseline returns the age and sex-based VO2 Max baseline
func GetVO2MaxBaseline(age int, sex string) float64 {
	var baseline float64

	switch {
	case age < 30:
		baseline = 38.0
	case age < 40:
		baseline = 36.0
	case age < 50:
		baseline = 33.0
	case age < 60:
		baseline = 30.0
	case age < 70:
		baseline = 27.0
	default:
		baseline = 24.0
	}

	if sex == "male" {
		baseline += 6.0
	}

	return baseline
}

// GetReactionTimeBaseline returns the age-based reaction time baseline in ms
func GetReactionTimeBaseline(age int) int {
	switch {
	case age < 20:
		return 200
	case age < 30:
		return 220
	case age < 40:
		return 240
	case age < 50:
		return 260
	case age < 60:
		return 280
	case age < 70:
		return 300
	default:
		return 320
	}
}
