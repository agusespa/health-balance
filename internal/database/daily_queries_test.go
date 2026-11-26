package database

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"health-balance/internal/models"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Init(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	})
	return db
}

func stripUpdated(ms []models.DailyMetric) []models.DailyMetric {
	out := make([]models.DailyMetric, len(ms))
	for i, m := range ms {
		m.UpdatedAt = time.Time{}
		out[i] = m
	}
	return out
}

func TestSaveIngestIsIdempotent(t *testing.T) {
	db := openTestDB(t)
	values := []models.DailyMetric{
		{Date: "2026-11-24", Metric: "rhr", Value: 56, Samples: 1, Source: "watch"},
		{Date: "2026-11-25", Metric: "rhr", Value: 55, Samples: 1, Source: "watch"},
		{Date: "2026-11-25", Metric: "steps", Value: 7400, Samples: 2, Source: "watch"},
	}
	run := models.IngestRun{Kind: models.IngestKindShortcut, Rows: 10, Accepted: 3, FromDate: "2026-11-24", ToDate: "2026-11-25"}

	if _, err := db.SaveIngest(run, values, nil); err != nil {
		t.Fatalf("first save: %v", err)
	}
	first, err := db.GetDailyMetrics("2026-11-01", "2026-11-30")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.SaveIngest(run, values, nil); err != nil {
		t.Fatalf("second save: %v", err)
	}
	second, err := db.GetDailyMetrics("2026-11-01", "2026-11-30")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stripUpdated(first), stripUpdated(second)) || len(second) != 3 {
		t.Errorf("re-sending changed the rows:\n%v\n%v", first, second)
	}
}

func TestSaveIngestOverwritesOnlyGivenValues(t *testing.T) {
	db := openTestDB(t)
	if _, err := db.SaveIngest(models.IngestRun{Kind: models.IngestKindShortcut}, []models.DailyMetric{
		{Date: "2026-11-25", Metric: "rhr", Value: 58, Samples: 1, Source: "watch"},
		{Date: "2026-11-25", Metric: "vo2_max", Value: 46.1, Samples: 1, Source: "watch"},
	}, nil); err != nil {
		t.Fatal(err)
	}
	// A later run replaces today's provisional RHR and has no VO2 reading.
	if _, err := db.SaveIngest(models.IngestRun{Kind: models.IngestKindShortcut}, []models.DailyMetric{
		{Date: "2026-11-25", Metric: "rhr", Value: 55, Samples: 1, Source: "watch"},
	}, nil); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetDailyMetrics("2026-11-25", "2026-11-25")
	if err != nil {
		t.Fatal(err)
	}
	want := []models.DailyMetric{
		{Date: "2026-11-25", Metric: "rhr", Value: 55, Samples: 1, Source: "watch"},
		{Date: "2026-11-25", Metric: "vo2_max", Value: 46.1, Samples: 1, Source: "watch"},
	}
	if !reflect.DeepEqual(stripUpdated(got), want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestGetDailyMetricsFilters(t *testing.T) {
	db := openTestDB(t)
	var values []models.DailyMetric
	for _, d := range []string{"2026-11-20", "2026-11-21", "2026-11-22"} {
		values = append(values,
			models.DailyMetric{Date: d, Metric: "rhr", Value: 55},
			models.DailyMetric{Date: d, Metric: "steps", Value: 9000},
			models.DailyMetric{Date: d, Metric: "hrv_sdnn", Value: 45},
		)
	}
	if _, err := db.SaveIngest(models.IngestRun{Kind: models.IngestKindBackfill}, values, nil); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetDailyMetrics("2026-11-21", "2026-11-22", "rhr", "steps")
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, m := range got {
		keys = append(keys, m.Date+" "+m.Metric)
		if m.UpdatedAt.IsZero() {
			t.Errorf("%s %s: updated_at not set", m.Date, m.Metric)
		}
	}
	want := []string{"2026-11-21 rhr", "2026-11-21 steps", "2026-11-22 rhr", "2026-11-22 steps"}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("got %v, want %v", keys, want)
	}
}

func TestGetLastIngestRun(t *testing.T) {
	db := openTestDB(t)
	if r, err := db.GetLastIngestRun(models.IngestKindShortcut); err != nil || r != nil {
		t.Fatalf("empty db: got %v, %v", r, err)
	}

	t0 := time.Date(2026, 11, 24, 6, 30, 0, 0, time.UTC)
	for i, kind := range []string{models.IngestKindShortcut, models.IngestKindShortcut, models.IngestKindGym} {
		run := models.IngestRun{Kind: kind, ReceivedAt: t0.Add(time.Duration(i) * time.Hour), Rows: i}
		if _, err := db.SaveIngest(run, nil, nil); err != nil {
			t.Fatal(err)
		}
	}

	r, err := db.GetLastIngestRun(models.IngestKindShortcut)
	if err != nil {
		t.Fatal(err)
	}
	if r == nil || r.Rows != 1 || !r.ReceivedAt.Equal(t0.Add(time.Hour)) {
		t.Errorf("got %+v, want the second shortcut run", r)
	}
}

func TestInsights(t *testing.T) {
	db := openTestDB(t)
	if in, err := db.GetLatestInsight(); err != nil || in != nil {
		t.Fatalf("empty db: got %v, %v", in, err)
	}
	for _, in := range []models.Insight{
		{WeekEnd: "2026-11-14", Model: "m", Markdown: "old"},
		{WeekEnd: "2026-11-21", Model: "m", Markdown: "first"},
		{WeekEnd: "2026-11-21", Model: "m2", Markdown: "regenerated"},
	} {
		if err := db.SaveInsight(in); err != nil {
			t.Fatal(err)
		}
	}
	in, err := db.GetLatestInsight()
	if err != nil {
		t.Fatal(err)
	}
	if in.WeekEnd != "2026-11-21" || in.Markdown != "regenerated" || in.Model != "m2" || in.CreatedAt.IsZero() {
		t.Errorf("got %+v", in)
	}
}
