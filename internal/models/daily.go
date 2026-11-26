package models

import "time"

// DailyMetric is one automatically collected value for one day.
type DailyMetric struct {
	Date      string    `json:"date"`   // YYYY-MM-DD
	Metric    string    `json:"metric"` // see ingest.Metric* constants
	Value     float64   `json:"value"`
	Samples   int       `json:"samples"`
	Source    string    `json:"source"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DayMetric names one stored daily value.
type DayMetric struct {
	Date   string
	Metric string
}

// Ingest run kinds.
const (
	IngestKindShortcut = "shortcut"
	IngestKindBackfill = "backfill"
	IngestKindGym      = "gym"
)

// IngestRun records one delivery of data, for "last sync" and staleness checks.
type IngestRun struct {
	Id         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	Kind       string    `json:"kind"`
	Rows       int       `json:"rows"`     // input lines or records
	Accepted   int       `json:"accepted"` // daily values written
	Rejected   int       `json:"rejected"` // input lines that could not be used
	FromDate   string    `json:"from_date"`
	ToDate     string    `json:"to_date"`
}

// Insight is the cached Claude summary for one week.
type Insight struct {
	WeekEnd   string // Friday that closes the week
	Model     string
	CreatedAt time.Time
	Markdown  string
}
