package ingest

import (
	"testing"
	"time"
)

func night(t *testing.T, rows string) Night {
	t.Helper()
	samples, bad := Parse(rows)
	if len(bad) != 0 {
		t.Fatalf("bad lines: %v", bad)
	}
	nights := aggregateSleep(samples)
	if len(nights) != 1 {
		t.Fatalf("got %d nights, want 1: %+v", len(nights), nights)
	}
	return nights[0]
}

func TestSleepStagesAcrossMidnight(t *testing.T) {
	rows := `SLEEP;In Bed;;2026-11-19T22:45:00+01:00;2026-11-20T06:45:00+01:00;Agustin’s iPhone
SLEEP;Core;;2026-11-19T23:10:00+01:00;2026-11-20T01:00:00+01:00;Agustin’s Apple Watch
SLEEP;Deep;;2026-11-20T01:00:00+01:00;2026-11-20T02:00:00+01:00;Agustin’s Apple Watch
SLEEP;Awake;;2026-11-20T02:00:00+01:00;2026-11-20T02:20:00+01:00;Agustin’s Apple Watch
SLEEP;REM;;2026-11-20T02:20:00+01:00;2026-11-20T03:00:00+01:00;Agustin’s Apple Watch
SLEEP;Core;;2026-11-20T03:00:00+01:00;2026-11-20T06:30:00+01:00;Agustin’s Apple Watch`
	n := night(t, rows)

	if n.Date != "2026-11-20" || n.Source != SourceWatch {
		t.Errorf("got %s from %s, want 2026-11-20 from watch", n.Date, n.Source)
	}
	if want := 7 * time.Hour; n.Asleep != want { // 23:10–06:30 is 7:20, minus 20 min awake
		t.Errorf("asleep %v, want %v", n.Asleep, want)
	}
	if n.BedtimeMin != 310 { // 23:10 is 310 min after 18:00
		t.Errorf("bedtime %v, want 310", n.BedtimeMin)
	}
	if n.AwakeMin != 20 || n.Awakenings != 1 {
		t.Errorf("awake %v min / %d awakenings, want 20 / 1", n.AwakeMin, n.Awakenings)
	}
}

func TestSleepBedtimeAfterMidnight(t *testing.T) {
	n := night(t, "SLEEP;Core;;2026-11-20T00:30:00+01:00;2026-11-20T07:30:00+01:00;Agustin’s Apple Watch")
	if n.BedtimeMin != 390 { // 00:30 is 6.5 h after 18:00
		t.Errorf("bedtime %v, want 390", n.BedtimeMin)
	}
}

func TestSleepIgnoresNapAndPrefersWatch(t *testing.T) {
	rows := `SLEEP;Asleep;;2026-11-19T23:00:00+01:00;2026-11-20T07:30:00+01:00;Agustin’s iPhone
SLEEP;Core;;2026-11-19T23:30:00+01:00;2026-11-20T06:30:00+01:00;Agustin’s Apple Watch
SLEEP;Core;;2026-11-20T14:00:00+01:00;2026-11-20T14:40:00+01:00;Agustin’s Apple Watch`
	n := night(t, rows)
	if n.Source != SourceWatch {
		t.Errorf("source %s, want watch even though the iPhone has more time", n.Source)
	}
	if n.Asleep != 7*time.Hour || n.Awakenings != 0 {
		t.Errorf("asleep %v / %d awakenings, want 7h / 0 (nap ignored)", n.Asleep, n.Awakenings)
	}
}

func TestSleepWithoutStagesUsesGaps(t *testing.T) {
	// Pre-2022 data: unspecified asleep segments, awake time is the gaps.
	rows := `SLEEP;Asleep;;2026-11-19T23:00:00+01:00;2026-11-20T03:00:00+01:00;Agustin’s iPhone
SLEEP;Asleep;;2026-11-20T03:30:00+01:00;2026-11-20T07:00:00+01:00;Agustin’s iPhone`
	n := night(t, rows)
	if n.Asleep != 7*time.Hour+30*time.Minute || n.AwakeMin != 30 || n.Awakenings != 1 {
		t.Errorf("got %+v", n)
	}
}

func TestSleepScore(t *testing.T) {
	good := Night{Asleep: 8 * time.Hour, BedtimeMin: 300, AwakeMin: 10, Awakenings: 1}
	tests := []struct {
		name  string
		n     Night
		usual float64
		want  float64
	}{
		{"full marks", good, 300, 100},
		{"no bedtime history", good, -1, 100},
		{"5 h asleep", Night{Asleep: 5 * time.Hour, BedtimeMin: 300}, 300, 12.5 + 30 + 20},
		{"45 min late", good, 255, 50 + 15 + 20},
		{"restless", Night{Asleep: 7 * time.Hour, BedtimeMin: 300, AwakeMin: 40, Awakenings: 5}, 300, 37.5 + 30 + 12},
		{"floors at zero", Night{Asleep: time.Hour, BedtimeMin: 600, AwakeMin: 200, Awakenings: 20}, 300, 0},
	}
	for _, tc := range tests {
		if got := SleepScore(tc.n, tc.usual); got != tc.want {
			t.Errorf("%s: SleepScore = %v, want %v", tc.name, got, tc.want)
		}
	}
}
