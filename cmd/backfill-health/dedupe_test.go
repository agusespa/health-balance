package main

import (
	"testing"

	"health-balance/internal/ingest"
)

func TestDedupeSources(t *testing.T) {
	rows := `STEPS;4000;count;2026-11-25T08:00:00+01:00;2026-11-25T09:00:00+01:00;Agustin’s iPhone
STEPS;3000;count;2026-11-25T12:00:00+01:00;2026-11-25T13:00:00+01:00;Agustin’s iPhone
STEPS;6500;count;2026-11-25T08:00:00+01:00;2026-11-25T13:00:00+01:00;Agustin’s Apple Watch
STEPS;900;count;2026-11-25T18:00:00+01:00;2026-11-25T19:00:00+01:00;Agustin’s Apple Watch
STEPS;5000;count;2026-11-26T08:00:00+01:00;2026-11-26T09:00:00+01:00;Agustin’s iPhone
HRV;40;ms;2026-11-25T02:00:00+01:00;2026-11-25T02:01:00+01:00;Agustin’s Apple Watch
HRV;50;ms;2026-11-25T03:00:00+01:00;2026-11-25T03:01:00+01:00;Other app
SLEEP;Asleep;;2026-11-24T23:00:00+01:00;2026-11-25T07:30:00+01:00;Agustin’s iPhone
SLEEP;In Bed;;2026-11-24T22:30:00+01:00;2026-11-25T07:30:00+01:00;Agustin’s iPhone
SLEEP;Core;;2026-11-24T23:30:00+01:00;2026-11-25T06:30:00+01:00;Agustin’s Apple Watch`
	samples, bad := ingest.Parse(rows)
	if len(bad) != 0 {
		t.Fatal(bad)
	}
	got := map[string]float64{}
	for _, dv := range ingest.Aggregate(dedupeSources(samples), ingest.Options{}) {
		got[dv.Date+" "+dv.Metric] = dv.Value
	}
	checks := map[string]float64{
		"2026-11-25 steps":       7400, // Watch 7400 beats iPhone 7000
		"2026-11-26 steps":       5000, // only the iPhone that day
		"2026-11-25 hrv_sdnn":    45,   // averages are left alone
		"2026-11-25 sleep_hours": 7,    // the Watch's night, though the iPhone has more
	}
	for k, want := range checks {
		if got[k] != want {
			t.Errorf("%s = %v, want %v", k, got[k], want)
		}
	}
}
