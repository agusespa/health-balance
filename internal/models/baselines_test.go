package models

import "testing"

func TestGetVO2MaxBaseline(t *testing.T) {
	testCases := []struct {
		name     string
		age      int
		sex      string
		expected float64
	}{
		{"Young female", 25, "female", 38.0},
		{"Young male", 25, "male", 44.0}, // 38.0 + 6.0
		{"Middle-aged female", 45, "female", 33.0},
		{"Middle-aged male", 45, "male", 39.0}, // 33.0 + 6.0
		{"Senior female", 65, "female", 27.0},
		{"Senior male", 65, "male", 33.0}, // 27.0 + 6.0
		{"Over 70 female", 75, "female", 24.0},
		{"Over 70 male", 75, "male", 30.0},          // 24.0 + 6.0
		{"Case insensitive male", 35, "MALE", 42.0}, // 36.0 + 6.0
		{"Case insensitive female", 35, "FEMALE", 36.0},
		{"Default to female if not specified", 35, "other", 36.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetVO2MaxBaseline(tc.age, tc.sex)
			if result != tc.expected {
				t.Errorf("GetVO2MaxBaseline(%d, %s) = %f; expected %f", tc.age, tc.sex, result, tc.expected)
			}
		})
	}
}

func TestGetReactionTimeBaseline(t *testing.T) {
	testCases := []struct {
		name     string
		age      int
		expected int
	}{
		{"Under 20", 19, 200},
		{"Exactly 20", 20, 220},
		{"Under 30", 29, 220},
		{"Exactly 30", 30, 240},
		{"Under 40", 39, 240},
		{"Exactly 40", 40, 260},
		{"Under 50", 49, 260},
		{"Exactly 50", 50, 280},
		{"Under 60", 59, 280},
		{"Exactly 60", 60, 300},
		{"Under 70", 69, 300},
		{"Exactly 70", 70, 320},
		{"Over 70", 75, 320},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetReactionTimeBaseline(tc.age)
			if result != tc.expected {
				t.Errorf("GetReactionTimeBaseline(%d) = %d; expected %d", tc.age, result, tc.expected)
			}
		})
	}
}
