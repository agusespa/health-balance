package services

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"health-balance/internal/database"
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
	if len(steps) != 1 || steps[0].Value != 9700 || steps[0].Source != "watch" {
		t.Errorf("steps %v, want 9700 from watch", steps)
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
