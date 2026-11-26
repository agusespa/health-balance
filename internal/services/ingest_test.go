package services

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"health-balance/internal/database"
	"health-balance/internal/ingest"
	"health-balance/internal/models"
	"health-balance/internal/testutil"
)

// A real Shortcut payload (RHR only so far) plus steps, to exercise the
// oldest-day rule.
const shortcutRows = `RHR; 53;count/min;2026-11-18T00:03:14+01:00;2026-11-18T23:54:13+01:00;Agustin’s Apple Watch
RHR; 55;count/min;2026-11-19T00:01:30+01:00;2026-11-19T23:54:46+01:00;Agustin’s Apple Watch
RHR; 55;count/min;2026-11-25T00:01:37+01:00;2026-11-25T20:18:32+01:00;Agustin’s Apple Watch
STEPS;3100;count;2026-11-18T18:00:00+01:00;2026-11-18T23:59:00+01:00;Agustin’s iPhone
STEPS;9500;count;2026-11-19T00:00:00+01:00;2026-11-19T23:59:00+01:00;Agustin’s iPhone
STEPS;9700;count;2026-11-19T00:00:00+01:00;2026-11-19T23:59:00+01:00;Agustin’s Apple Watch
BOGUS;1;count;2026-11-19T00:00:00+01:00;2026-11-19T23:59:00+01:00;Watch`

func TestIngestShortcutRowsStoresAndIsIdempotent(t *testing.T) {
	db, err := database.Init(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, 11, 25, 20, 23, 0, 0, time.UTC)
	res, err := IngestShortcutRows(db, shortcutRows, now)
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if res.Rows != 7 || res.Rejected != 1 || res.Accepted != 4 || res.From != "2026-11-18" || res.To != "2026-11-25" {
		t.Errorf("result %+v", res)
	}
	if want := map[string]int{"rhr": 3, "steps": 1}; !reflect.DeepEqual(res.Metrics, want) {
		t.Errorf("metrics %v, want %v (steps on the oldest day skipped)", res.Metrics, want)
	}

	first, _ := db.GetDailyMetrics("2026-11-01", "2026-11-30")
	if _, err := IngestShortcutRows(db, shortcutRows, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	second, _ := db.GetDailyMetrics("2026-11-01", "2026-11-30")
	if len(first) != len(second) {
		t.Fatalf("row count changed: %d → %d", len(first), len(second))
	}
	for i := range first {
		first[i].UpdatedAt, second[i].UpdatedAt = time.Time{}, time.Time{}
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("re-sending changed values:\n%v\n%v", first, second)
	}

	steps, _ := db.GetDailyMetrics("2026-11-19", "2026-11-19", "steps")
	if len(steps) != 1 || steps[0].Value != 19200 || steps[0].Source != "mixed" {
		t.Errorf("steps %v, want 9500 + 9700 from both sources", steps)
	}

	run, _ := db.GetLastIngestRun(models.IngestKindShortcut)
	if run == nil || !run.ReceivedAt.Equal(now.Add(time.Hour)) || run.Accepted != 4 {
		t.Errorf("last run %+v", run)
	}
}

func TestIngestShortcutRowsRejectsUnusablePayload(t *testing.T) {
	res, err := IngestShortcutRows(&testutil.MockDB{}, "nonsense\n\nRHR;x;count/min;bad;bad;Watch", time.Now())
	if !errors.Is(err, ErrNoUsableRows) {
		t.Fatalf("err = %v, want ErrNoUsableRows", err)
	}
	if res.Rows != 2 || res.Rejected != 2 || len(res.Bad) != 2 {
		t.Errorf("result %+v", res)
	}
}

func TestIngestShortcutRowsClearsDeletedDiaryDays(t *testing.T) {
	db, err := database.Init(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// First delivery: food on the 20th–22nd, water on the 21st.
	first := `KCAL;300;kcal;2026-11-19T08:00:00+01:00;2026-11-19T08:00:00+01:00;Cronometer
KCAL;2100;kcal;2026-11-20T08:00:00+01:00;2026-11-20T08:00:00+01:00;Cronometer
KCAL;900;kcal;2026-11-21T08:00:00+01:00;2026-11-21T08:00:00+01:00;Cronometer
PROTEIN;60;g;2026-11-21T08:00:00+01:00;2026-11-21T08:00:00+01:00;Cronometer
WATER;1500;mL;2026-11-21T08:00:00+01:00;2026-11-21T08:00:00+01:00;Cronometer
KCAL;2300;kcal;2026-11-22T08:00:00+01:00;2026-11-22T08:00:00+01:00;Cronometer`
	if _, err := IngestShortcutRows(db, first, time.Now()); err != nil {
		t.Fatal(err)
	}

	// The 21st was deleted in the diary and is no longer in Health. A step
	// line stretches the delivery to the 23rd, which had nothing logged.
	second := `KCAL;2100;kcal;2026-11-20T08:00:00+01:00;2026-11-20T08:00:00+01:00;Cronometer
KCAL;2300;kcal;2026-11-22T08:00:00+01:00;2026-11-22T08:00:00+01:00;Cronometer
STEPS;5000;count;2026-11-23T08:00:00+01:00;2026-11-23T09:00:00+01:00;Agustin’s iPhone`
	res, err := IngestShortcutRows(db, second, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if res.Cleared != 2 {
		t.Errorf("cleared %d, want 2 (energy on the 21st and 23rd; protein and water weren't sent)", res.Cleared)
	}

	got := map[string]float64{}
	rows, _ := db.GetDailyMetrics("2026-11-01", "2026-11-30")
	for _, r := range rows {
		got[r.Date+" "+r.Metric] = r.Value
	}
	if _, ok := got["2026-11-21 energy_in_kcal"]; ok {
		t.Error("deleted day's energy is still stored")
	}
	if got["2026-11-20 energy_in_kcal"] != 2100 || got["2026-11-22 energy_in_kcal"] != 2300 {
		t.Errorf("kept days changed: %v", got)
	}
	// Keys missing from the whole delivery are left alone.
	if got["2026-11-21 protein_g"] != 60 || got["2026-11-21 water_ml"] != 1500 {
		t.Errorf("protein/water should be untouched: %v", got)
	}
}

func TestStaleDiaryDaysOnlyInsideEachKeysWindow(t *testing.T) {
	// Water is sent for 2 days, food for 7: water days before its window
	// must not be cleared.
	rows := `KCAL;2100;kcal;2026-11-19T08:00:00+01:00;2026-11-19T08:00:00+01:00;Cronometer
KCAL;2200;kcal;2026-11-21T08:00:00+01:00;2026-11-21T08:00:00+01:00;Cronometer
WATER;1500;mL;2026-11-24T08:00:00+01:00;2026-11-24T08:00:00+01:00;Cronometer
KCAL;2300;kcal;2026-11-25T08:00:00+01:00;2026-11-25T08:00:00+01:00;Cronometer`
	samples, _ := ingest.Parse(rows)
	daily := ingest.Aggregate(samples, ingest.Options{SkipOldestDay: true})
	got := map[string]bool{}
	for _, c := range staleDiaryDays(samples, daily, "2026-11-25") {
		got[c.Date+" "+c.Metric] = true
	}
	want := map[string]bool{
		"2026-11-20 energy_in_kcal": true, "2026-11-22 energy_in_kcal": true,
		"2026-11-23 energy_in_kcal": true, "2026-11-24 energy_in_kcal": true,
		"2026-11-25 water_ml": true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("cleared %v\nwant    %v", got, want)
	}
}

func TestIngestShortcutRowsTreatsZeroDiaryTotalsAsEmpty(t *testing.T) {
	db, err := database.Init(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := IngestShortcutRows(db, `KCAL;100;kcal;2026-11-23T00:00:00+01:00;2026-11-23T00:00:00+01:00;
KCAL;2100;kcal;2026-11-24T00:00:00+01:00;2026-11-24T00:00:00+01:00;`, time.Now()); err != nil {
		t.Fatal(err)
	}
	// Grouped by day: the 24th was emptied in the diary and comes as 0.
	rows := `KCAL;0;kcal;2026-11-23T00:00:00+01:00;2026-11-23T00:00:00+01:00;
KCAL;0;kcal;2026-11-24T00:00:00+01:00;2026-11-24T00:00:00+01:00;
KCAL;900;kcal;2026-11-25T00:00:00+01:00;2026-11-25T00:00:00+01:00;
WATER;0;mL;2026-11-24T00:00:00+01:00;2026-11-24T00:00:00+01:00;
WATER;1500;mL;2026-11-25T00:00:00+01:00;2026-11-25T00:00:00+01:00;`
	if _, err := IngestShortcutRows(db, rows, time.Now()); err != nil {
		t.Fatal(err)
	}
	got, _ := db.GetDailyMetrics("2026-11-01", "2026-11-30")
	var keys []string
	for _, r := range got {
		keys = append(keys, r.Date+" "+r.Metric)
	}
	// The 24th's 2100 kcal is cleared by its 0; no zeros are stored.
	want := []string{"2026-11-25 energy_in_kcal", "2026-11-25 water_ml"}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("stored %v, want %v", keys, want)
	}
}
