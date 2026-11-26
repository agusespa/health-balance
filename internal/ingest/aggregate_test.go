package ingest

import (
	"math"
	"reflect"
	"testing"
)

func values(t *testing.T, rows string, opt Options) map[string]DailyValue {
	t.Helper()
	samples, bad := Parse(rows)
	if len(bad) != 0 {
		t.Fatalf("bad lines: %v", bad)
	}
	out := map[string]DailyValue{}
	for _, dv := range Aggregate(samples, opt) {
		out[dv.Date+" "+dv.Metric] = dv
	}
	return out
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestAggregateLastMeanAndMaxSource(t *testing.T) {
	rows := `RHR;58;count/min;2026-11-25T00:01:37+01:00;2026-11-25T08:00:00+01:00;Agustin’s Apple Watch
RHR;55;count/min;2026-11-25T00:01:37+01:00;2026-11-25T20:18:32+01:00;Agustin’s Apple Watch
HRV;40;ms;2026-11-25T02:00:00+01:00;2026-11-25T02:01:00+01:00;Agustin’s Apple Watch
HRV;50;ms;2026-11-25T09:00:00+01:00;2026-11-25T09:01:00+01:00;Agustin’s Apple Watch
STEPS;4000;count;2026-11-25T08:00:00+01:00;2026-11-25T09:00:00+01:00;Agustin’s iPhone
STEPS;3000;count;2026-11-25T12:00:00+01:00;2026-11-25T13:00:00+01:00;Agustin’s iPhone
STEPS;6500;count;2026-11-25T08:00:00+01:00;2026-11-25T13:00:00+01:00;Agus' Apple Watch
STEPS;900;count;2026-11-25T18:00:00+01:00;2026-11-25T19:00:00+01:00;Agustin’s Apple Watch
BPSYS;118;mmHg;2026-11-25T07:00:00+01:00;2026-11-25T07:00:00+01:00;Omron
BPSYS;114;mmHg;2026-11-25T07:05:00+01:00;2026-11-25T07:05:00+01:00;Omron`
	v := values(t, rows, Options{})

	if got := v["2026-11-25 rhr"]; got.Value != 55 || got.Samples != 2 {
		t.Errorf("rhr = %+v, want the later 55", got)
	}
	if got := v["2026-11-25 hrv_sdnn"]; !near(got.Value, 45) {
		t.Errorf("hrv = %v, want mean 45", got.Value)
	}
	// Watch 6500+900 = 7400 beats iPhone 4000+3000 = 7000; both Watch names merge.
	if got := v["2026-11-25 steps"]; got.Value != 7400 || got.Source != SourceWatch || got.Samples != 2 {
		t.Errorf("steps = %+v, want 7400 from watch", got)
	}
	if got := v["2026-11-25 bp_sys"]; !near(got.Value, 116) || got.Source != "Omron" {
		t.Errorf("bp_sys = %+v, want 116 from Omron", got)
	}
}

func TestAggregateDayUsesSampleOffset(t *testing.T) {
	// 23:30 local on the 24th is already the 25th in UTC; it stays on the 24th.
	v := values(t, "STEPS;100;count;2026-11-24T23:30:00+01:00;2026-11-24T23:40:00+01:00;Agustin’s iPhone", Options{})
	if _, ok := v["2026-11-24 steps"]; !ok {
		t.Errorf("want steps on 2026-11-24, got %v", v)
	}
}

func TestAggregateSkipOldestDay(t *testing.T) {
	rows := `RHR;53;count/min;2026-11-18T00:03:14+01:00;2026-11-18T23:54:13+01:00;Agustin’s Apple Watch
STEPS;2000;count;2026-11-18T20:00:00+01:00;2026-11-18T21:00:00+01:00;Agustin’s Apple Watch
STEPS;9000;count;2026-11-19T08:00:00+01:00;2026-11-19T21:00:00+01:00;Agustin’s Apple Watch
SLEEP;Core;;2026-11-18T03:00:00+01:00;2026-11-18T06:00:00+01:00;Agustin’s Apple Watch
SLEEP;Core;;2026-11-18T23:00:00+01:00;2026-11-19T06:00:00+01:00;Agustin’s Apple Watch`
	v := values(t, rows, Options{SkipOldestDay: true})

	if _, ok := v["2026-11-18 rhr"]; !ok {
		t.Error("point metric rhr on the oldest day should be kept")
	}
	if _, ok := v["2026-11-18 steps"]; ok {
		t.Error("steps on the oldest day should be skipped")
	}
	if _, ok := v["2026-11-18 sleep_hours"]; ok {
		t.Error("the oldest night should be skipped")
	}
	if got := v["2026-11-19 sleep_hours"]; got.Value != 7 {
		t.Errorf("night of 2026-11-19 = %v h, want 7", got.Value)
	}
	if got := v["2026-11-19 steps"]; got.Value != 9000 {
		t.Errorf("steps 2026-11-19 = %v, want 9000", got.Value)
	}
}

func TestAggregateMindfulDedupesSessions(t *testing.T) {
	rows := `MIND;0;min;2026-11-20T08:00:00+01:00;2026-11-20T08:05:00+01:00;Agustin’s Apple Watch
MIND;0;min;2026-11-20T08:00:00+01:00;2026-11-20T08:05:00+01:00;Agustin’s iPhone
MIND;0;min;2026-11-20T21:00:00+01:00;2026-11-20T21:10:00+01:00;Agustin’s Apple Watch`
	v := values(t, rows, Options{})
	if got := v["2026-11-20 mindful_sessions"]; got.Value != 2 {
		t.Errorf("sessions = %v, want 2", got.Value)
	}
	if got := v["2026-11-20 mindful_min"]; got.Value != 15 {
		t.Errorf("minutes = %v, want 15", got.Value)
	}
}

func TestAggregateIsDeterministic(t *testing.T) {
	rows := `STEPS;5000;count;2026-11-20T08:00:00+01:00;2026-11-20T09:00:00+01:00;Agustin’s iPhone
STEPS;5000;count;2026-11-20T08:00:00+01:00;2026-11-20T09:00:00+01:00;Agustin’s Apple Watch
HRV;40;ms;2026-11-20T02:00:00+01:00;2026-11-20T02:01:00+01:00;Agustin’s Apple Watch
RHR;50;count/min;2026-11-20T00:00:00+01:00;2026-11-20T23:00:00+01:00;Agustin’s Apple Watch`
	samples, _ := Parse(rows)
	first := Aggregate(samples, Options{})
	for i := 0; i < 20; i++ {
		if got := Aggregate(samples, Options{}); !reflect.DeepEqual(got, first) {
			t.Fatalf("run %d differs:\n%v\n%v", i, got, first)
		}
	}
	if first[len(first)-1].Metric != MetricSteps || first[len(first)-1].Source != SourcePhone {
		t.Errorf("tie between sources should pick the alphabetically first: %+v", first[len(first)-1])
	}
}

func TestAggregateWorkingSets(t *testing.T) {
	// n8n adds one SETS line per day, including explicit zeros, and may split
	// a day into several workouts.
	rows := `SETS;12;count;2026-11-24T00:00:00+01:00;2026-11-24T23:59:59+01:00;gympass
SETS;6;count;2026-11-24T00:00:00+01:00;2026-11-24T23:59:59+01:00;gympass
SETS;0;count;2026-11-25T00:00:00+01:00;2026-11-25T23:59:59+01:00;gympass
SETS;9;;2026-11-26T00:00:00+01:00;2026-11-26T23:59:59+01:00;gympass`
	v := values(t, rows, Options{})
	want := map[string]float64{"2026-11-24": 18, "2026-11-25": 0, "2026-11-26": 9}
	for date, sets := range want {
		got, ok := v[date+" working_sets"]
		if !ok || got.Value != sets || got.Source != "gympass" {
			t.Errorf("%s: got %+v (present %v), want %v sets from gympass", date, got, ok, sets)
		}
	}
}
