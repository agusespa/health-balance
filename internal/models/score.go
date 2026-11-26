package models

// Metric kinds.
const (
	KindMarker    = "marker"    // slow-moving health markers, carried forward when missing
	KindBehaviour = "behaviour" // habits, scored on a 4-week average
)

// Metric statuses for one week.
const (
	StatusMeasured = "measured" // data from this week
	StatusCarried  = "carried"  // last value reused unchanged
	StatusImputed  = "imputed"  // last value fading toward neutral
	StatusMissing  = "missing"  // no data yet; contributes 0
)

// MetricScore is one metric's contribution to a week's score.
type MetricScore struct {
	Metric        string  `json:"metric"`
	Label         string  `json:"label"`
	Kind          string  `json:"kind"`
	Status        string  `json:"status"`
	Value         float64 `json:"value"`
	Reference     float64 `json:"reference"`
	ValueText     string  `json:"value_text"`
	ReferenceText string  `json:"reference_text"`
	Points        float64 `json:"points"`
}

// WeeklyScore is the Biological Reserve Score after one Saturday–Friday week.
type WeeklyScore struct {
	WeekEnd  string        `json:"week_end"`
	Score    float64       `json:"score"`
	Change   float64       `json:"change"`
	AgingTax float64       `json:"aging_tax"`
	Target   float64       `json:"target"`
	Metrics  []MetricScore `json:"metrics"`
}
