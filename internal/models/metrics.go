package models

// UserProfile stores user-specific data for calculations
type UserProfile struct {
	Id        int     `json:"-"`
	BirthDate string  `json:"birth_date"`
	Sex       string  `json:"sex"`
	HeightCm  float64 `json:"height_cm"`
}
