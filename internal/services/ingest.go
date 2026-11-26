package services

import (
	"errors"
	"fmt"
	"time"

	"health-balance/internal/database"
	"health-balance/internal/ingest"
	"health-balance/internal/models"
)

// ErrNoUsableRows means a payload had nothing that could be stored.
var ErrNoUsableRows = errors.New("no usable rows")

// IngestResult summarises one Shortcut delivery.
type IngestResult struct {
	RunId    int64            `json:"run_id"`
	Rows     int              `json:"rows"`     // non-blank input lines
	Accepted int              `json:"accepted"` // daily values written
	Rejected int              `json:"rejected"` // lines that could not be used
	From     string           `json:"from"`
	To       string           `json:"to"`
	Cleared  int              `json:"cleared"` // stale diary values deleted
	Metrics  map[string]int   `json:"metrics"` // daily values written per metric
	Bad      []ingest.BadLine `json:"bad"`     // first few rejected lines
}

const maxBadLinesReported = 10

// IngestShortcutRows parses the Shortcut's rows, reduces them to daily values
// and stores them. The oldest day of the payload is skipped for totals and
// averages (it may be partial and was stored complete by an earlier run).
// Sending the same rows again leaves the stored values unchanged.
func IngestShortcutRows(db database.Querier, rows string, receivedAt time.Time) (IngestResult, error) {
	samples, bad := ingest.Parse(rows)
	res := IngestResult{
		Rows:     len(samples) + len(bad),
		Rejected: len(bad),
		Metrics:  map[string]int{},
		Bad:      bad,
	}
	if res.Bad == nil {
		res.Bad = []ingest.BadLine{}
	}
	if len(res.Bad) > maxBadLinesReported {
		res.Bad = res.Bad[:maxBadLinesReported]
	}
	if len(samples) == 0 {
		return res, ErrNoUsableRows
	}

	for _, s := range samples {
		d := s.Start.Format("2006-01-02")
		if res.From == "" || d < res.From {
			res.From = d
		}
		if d > res.To {
			res.To = d
		}
	}

	daily := ingest.Aggregate(samples, ingest.Options{SkipOldestDay: true})
	values := make([]models.DailyMetric, len(daily))
	for i, dv := range daily {
		values[i] = models.DailyMetric{Date: dv.Date, Metric: dv.Metric, Value: dv.Value, Samples: dv.Samples, Source: dv.Source}
		res.Metrics[dv.Metric]++
	}
	res.Accepted = len(values)
	clear := staleDiaryDays(samples, daily, res.To)
	res.Cleared = len(clear)

	id, err := db.SaveIngest(models.IngestRun{
		ReceivedAt: receivedAt,
		Kind:       models.IngestKindShortcut,
		Rows:       res.Rows,
		Accepted:   res.Accepted,
		Rejected:   res.Rejected,
		FromDate:   res.From,
		ToDate:     res.To,
	}, values, clear)
	if err != nil {
		return res, fmt.Errorf("save ingest: %w", err)
	}
	res.RunId = id
	return res, nil
}

// staleDiaryDays lists the stored food diary values to delete: for each diary
// key the delivery contains at all, every day after that key's oldest (which
// is skipped anyway) up to the delivery's newest day that got no value. Deleting a day's entries
// in the diary removes them from Health, and they then stop arriving, so
// without this the old value would stay. A delivery without a key at all (a
// sender that doesn't read it) clears nothing for that key.
func staleDiaryDays(samples []ingest.Sample, daily []ingest.DailyValue, to string) []models.DayMetric {
	oldest := ingest.OldestDays(samples)
	have := map[models.DayMetric]bool{}
	for _, dv := range daily {
		have[models.DayMetric{Date: dv.Date, Metric: dv.Metric}] = true
	}
	end, err := time.Parse("2006-01-02", to)
	if err != nil {
		return nil
	}
	var out []models.DayMetric
	for _, key := range ingest.DiaryKeys {
		from, ok := oldest[key]
		if !ok {
			continue
		}
		start, err := time.Parse("2006-01-02", from)
		if err != nil {
			continue
		}
		metric, _ := ingest.MetricFor(key)
		for d := start.AddDate(0, 0, 1); !d.After(end); d = d.AddDate(0, 0, 1) {
			dm := models.DayMetric{Date: d.Format("2006-01-02"), Metric: metric}
			if !have[dm] {
				out = append(out, dm)
			}
		}
	}
	return out
}
