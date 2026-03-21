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
	SleepScore     int
	WaistCm        float64
	BodyWeightKg   float64
	RHR            int
	SystolicBP     int
	DiastolicBP    int
	NutritionScore float64
}

// FitnessMetrics represents the Fitness Pillar
type FitnessMetrics struct {
	Date            string
	VO2Max          float64
	Workouts        int
	DailySteps      int
	Mobility        int
	CardioRecovery  int
	LowerBodyWeight float64
	LowerBodyReps   int
	DeadHangSeconds int
}

// CognitionMetrics represents the Cognition Pillar
type CognitionMetrics struct {
	Date         string
	Mindfulness  int
	DeepLearning int
	StressScore  int
	SocialDays   int
}

// UserProfile stores user-specific data for calculations
type UserProfile struct {
	Id        int
	BirthDate string
	Sex       string
	HeightCm  float64
}
