package models

// GetVO2MaxBaseline returns the age and sex-based VO2 Max baseline
// Based on standard fitness norms
func GetVO2MaxBaseline(age int, sex string) float64 {
	if sex == "male" {
		switch {
		case age < 30:
			return 44.0
		case age < 40:
			return 42.0
		case age < 50:
			return 39.0
		case age < 60:
			return 36.0
		case age < 70:
			return 33.0
		default:
			return 30.0
		}
	} else { // female
		switch {
		case age < 30:
			return 38.0
		case age < 40:
			return 36.0
		case age < 50:
			return 33.0
		case age < 60:
			return 30.0
		case age < 70:
			return 27.0
		default:
			return 24.0
		}
	}
}

// GetReactionTimeBaseline returns the age-based reaction time baseline in ms
// Based on human performance research
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
