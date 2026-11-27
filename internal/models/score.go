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
	// MaxPoints is the most this metric can add to the target.
	MaxPoints float64 `json:"max_points"`
	// WeeksOld is how many weeks ago the value was last measured, for a
	// carried or imputed value; 0 when measured this week or never.
	WeeksOld int `json:"weeks_old,omitempty"`
}

// WeeklyScore is the Biological Reserve Score after one Saturday–Friday week.
type WeeklyScore struct {
	WeekEnd  string  `json:"week_end"`
	Score    float64 `json:"score"`
	Change   float64 `json:"change"`
	AgingTax float64 `json:"aging_tax"`
	Target   float64 `json:"target"`
	// Settle is where the score would level off if every week were like this
	// one: below the target, because the aging tax keeps pulling it down.
	Settle  float64       `json:"settle"`
	Metrics []MetricScore `json:"metrics"`
}
