package models

// MasterScore represents the calculated overall longevity score
type MasterScore struct {
	Date           string
	Score          float64
	HealthScore    float64
	FitnessScore   float64
	CognitionScore float64
	AgingTax       float64
}

// HealthMetrics represents the Health Pillar
type HealthMetrics struct {
	Date           string
	SleepScore     int     // Weekly avg score (0-100)
	WaistCm        float64 // Waist circumference in cm
	RHR            int     // Resting Heart Rate
	NutritionScore float64 // Manual 1-10 score
}

// FitnessMetrics represents the Fitness Pillar
type FitnessMetrics struct {
	Date           string
	VO2Max         float64 // Current VO2 Max
	WeeklyWorkouts int
	DailySteps     int
	WeeklyMobility int
	CardioRecovery int // 60s BPM drop
}

// CognitionMetrics represents the Cognition Pillar
type CognitionMetrics struct {
	Date              string
	DualNBackLevel    int
	ReactionTime      int // Current in ms
	WeeklyMindfulness int
}

// UserProfile stores user-specific data for calculations
type UserProfile struct {
	Id        int
	BirthDate string  // YYYY-MM-DD format
	Sex       string  // "male" or "female"
	HeightCm  float64 // Height in centimeters
}
