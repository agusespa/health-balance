package services

import (
	"errors"
	"math"
	"path/filepath"
	"testing"
	"time"

	"health-balance/internal/database"
	"health-balance/internal/ingest"
	"health-balance/internal/models"
	"health-balance/internal/testutil"
)

var scoreProfile = models.UserProfile{BirthDate: "1990-06-15", Sex: "male", HeightCm: 180}

// Weeks end on Fridays: 2026-01-09, 01-16, 01-23, …
const firstFriday = "2026-01-09"

func scoreDB(t *testing.T, values []models.DailyMetric) *database.DB {
	t.Helper()
	db, err := database.Init(filepath.Join(t.TempDir(), "score.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.SaveIngest(models.IngestRun{Kind: models.IngestKindBackfill}, values, nil); err != nil {
		t.Fatal(err)
	}
	return db
}

// daily returns one value per day for n days ending on last (inclusive).
func daily(metric, last string, n int, value float64) []models.DailyMetric {
	end, _ := time.Parse(dateLayout, last)
	var out []models.DailyMetric
	for i := n - 1; i >= 0; i-- {
		out = append(out, models.DailyMetric{Date: end.AddDate(0, 0, -i).Format(dateLayout), Metric: metric, Value: value})
	}
	return out
}

func friday(i int) string {
	f, _ := time.Parse(dateLayout, firstFriday)
	return f.AddDate(0, 0, 7*i).Format(dateLayout)
}

func metricOf(t *testing.T, w models.WeeklyScore, metric string) models.MetricScore {
	t.Helper()
	for _, m := range w.Metrics {
		if m.Metric == metric {
			return m
		}
	}
	t.Fatalf("week %s has no %s", w.WeekEnd, metric)
	return models.MetricScore{}
}

func scores(t *testing.T, db *database.DB, weeks int) []models.WeeklyScore {
	t.Helper()
	out, err := ComputeWeeklyScores(db, scoreProfile, firstFriday, friday(weeks-1))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != weeks {
		t.Fatalf("got %d weeks, want %d", len(out), weeks)
	}
	return out
}

func TestScoreWithNoDataStaysNearBaseline(t *testing.T) {
	out := scores(t, scoreDB(t, nil), 2)
	w := out[0]
	if w.Target != 1000 {
		t.Errorf("target %v, want 1000 with no data", w.Target)
	}
	// 35 years old: tax = 1000 × 35² / 8000 / 52.
	tax := 1000 * 35.0 * 35.0 / 8000 / 52
	want := (1000 - tax) + tax*scoreAdjustmentRate
	if math.Abs(w.AgingTax-tax) > 1e-9 || math.Abs(w.Score-want) > 1e-9 || math.Abs(w.Change-(want-1000)) > 1e-9 {
		t.Errorf("week 1 = %+v, want score %v tax %v", w, want, tax)
	}
	for _, m := range w.Metrics {
		if m.Status != models.StatusMissing || m.Points != 0 {
			t.Errorf("%s: %s %v, want missing 0", m.Metric, m.Status, m.Points)
		}
	}
	if len(w.Metrics) != 11 {
		t.Errorf("%d metrics, want 11", len(w.Metrics))
	}
}

func TestMarkerCarriedThenFades(t *testing.T) {
	// One VO2 reading of 46 in week 0; the baseline at 35 for a man is 42.
	db := scoreDB(t, []models.DailyMetric{{Date: friday(0), Metric: ingest.MetricVO2Max, Value: 46}})
	out := scores(t, db, 7)

	measured := metricOf(t, out[0], "vo2_max")
	if measured.Status != models.StatusMeasured || measured.Points != 10 { // (46-42) × 2.5
		t.Fatalf("week 0: %+v", measured)
	}
	for i := 1; i <= 4; i++ {
		if m := metricOf(t, out[i], "vo2_max"); m.Status != models.StatusCarried || m.Points != 10 || m.WeeksOld != i {
			t.Errorf("week %d: %s %v, %d weeks old, want carried 10, %d weeks old", i, m.Status, m.Points, m.WeeksOld, i)
		}
	}
	for i, want := range map[int]float64{5: 7.5, 6: 5.625} {
		if m := metricOf(t, out[i], "vo2_max"); m.Status != models.StatusImputed || math.Abs(m.Points-want) > 1e-9 || m.WeeksOld != i {
			t.Errorf("week %d: %s %v, %d weeks old, want imputed %v", i, m.Status, m.Points, m.WeeksOld, want)
		}
	}
}

func TestRHRAgainstBaseline(t *testing.T) {
	values := daily(ingest.MetricRHR, "2026-01-02", 90, 55)               // the 90 days before week 0
	values = append(values, daily(ingest.MetricRHR, friday(0), 7, 53)...) // week 0
	out := scores(t, scoreDB(t, values), 1)
	m := metricOf(t, out[0], "rhr")
	if m.Reference != 55 || m.Value != 53 || m.Points != 2 {
		t.Errorf("rhr %+v, want 53 vs 55 = +2", m)
	}
}

func TestBehaviourAveragesAndFades(t *testing.T) {
	// 12,000 steps a day in weeks 0 and 1, then no data.
	values := daily(ingest.MetricSteps, friday(1), 14, 12000)
	out := scores(t, scoreDB(t, values), 4)

	want := []struct {
		status   string
		avg      float64
		weeksOld int
	}{
		{models.StatusMeasured, 12000, 0},
		{models.StatusMeasured, 12000, 0},
		{models.StatusCarried, 12000, 1},                 // reused once
		{models.StatusImputed, (36000 + 10000) / 4.0, 2}, // 8000 + 4000 × 0.5
	}
	for i, w := range want {
		m := metricOf(t, out[i], "steps")
		if m.Status != w.status || m.Value != w.avg || m.WeeksOld != w.weeksOld {
			t.Errorf("week %d: %s %v, %d weeks old, want %s %v, %d weeks old", i, m.Status, m.Value, m.WeeksOld, w.status, w.avg, w.weeksOld)
		}
	}
	if m := metricOf(t, out[0], "steps"); m.Points != 2 { // (12000-8000)/2000 × 1.0
		t.Errorf("points %v, want 2", m.Points)
	}
}

func TestBehaviourNeedsThreeDays(t *testing.T) {
	out := scores(t, scoreDB(t, daily(ingest.MetricSteps, friday(0), 2, 20000)), 1)
	if m := metricOf(t, out[0], "steps"); m.Status != models.StatusMissing {
		t.Errorf("2 days of steps: %s, want missing", m.Status)
	}
}

func TestWorkingSetsUnknownBeforeFirstLine(t *testing.T) {
	values := []models.DailyMetric{
		{Date: friday(2), Metric: ingest.MetricWorkingSets, Value: 8},
		{Date: friday(2), Metric: ingest.MetricWorkingSets + "_other", Value: 0}, // unrelated metric
	}
	values = append(values, daily(ingest.MetricWorkingSets, friday(3), 7, 2)...) // 14 a week
	out := scores(t, scoreDB(t, values), 4)

	for i := 0; i < 2; i++ {
		if m := metricOf(t, out[i], "working_sets"); m.Status != models.StatusMissing || m.Points != 0 {
			t.Errorf("week %d before gym logging: %+v", i, m)
		}
	}
	if m := metricOf(t, out[2], "working_sets"); m.Status != models.StatusMeasured || m.Value != 8 {
		t.Errorf("week 2: %+v", m)
	}
	if m := metricOf(t, out[3], "working_sets"); m.Value != 11 { // (8 + 14) / 2
		t.Errorf("week 3 average %v, want 11", m.Value)
	}
}

func TestMindfulnessZeroWhenSynced(t *testing.T) {
	values := daily(ingest.MetricRHR, friday(0), 7, 50)
	values = append(values, models.DailyMetric{Date: friday(0), Metric: ingest.MetricMindfulSessions, Value: 1})
	values = append(values, daily(ingest.MetricRHR, friday(1), 7, 50)...)
	out := scores(t, scoreDB(t, values), 2)

	if m := metricOf(t, out[0], "mindfulness"); m.Value != 1 || m.Status != models.StatusMeasured {
		t.Errorf("week 0: %+v", m)
	}
	// Week 1 synced with no sessions: a real zero, averaged with week 0.
	if m := metricOf(t, out[1], "mindfulness"); m.Value != 0.5 || m.Status != models.StatusMeasured {
		t.Errorf("week 1: %+v", m)
	}
}

func TestGetWeeklyScoresNeedsProfile(t *testing.T) {
	if _, err := GetWeeklyScores(&testutil.MockDB{}); !errors.Is(err, ErrProfileRequired) {
		t.Errorf("err = %v, want ErrProfileRequired", err)
	}
}
