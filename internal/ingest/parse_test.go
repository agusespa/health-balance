package ingest

import (
	"math"
	"testing"
)

func TestParseRealShortcutLine(t *testing.T) {
	samples, bad := Parse("RHR; 53;count/min;2026-11-18T00:03:14+01:00;2026-11-18T23:54:13+01:00;Agustin’s Apple Watch")
	if len(bad) != 0 {
		t.Fatalf("bad lines: %v", bad)
	}
	if len(samples) != 1 {
		t.Fatalf("got %d samples, want 1", len(samples))
	}
	s := samples[0]
	if s.Key != KeyRHR || s.Value != 53 || s.Source != "Agustin’s Apple Watch" {
		t.Errorf("got %+v", s)
	}
	if _, off := s.Start.Zone(); off != 3600 {
		t.Errorf("offset %d, want +01:00 kept", off)
	}
}

func TestParseValuesAndUnits(t *testing.T) {
	tests := []struct {
		line string
		key  string
		want float64
	}{
		{"VO2;48,2;mL/min·kg;2026-11-20T10:00:00+01:00;2026-11-20T10:00:00+01:00;Agus' Apple Watch", KeyVO2, 48.2},
		{"STEPS;12 345;count;2026-11-20T00:00:00+01:00;2026-11-20T23:59:59+01:00;Agustin’s iPhone", KeySteps, 12345},
		{"STEPS;12 345;count;2026-11-20T00:00:00+01:00;2026-11-20T23:59:59+01:00;Agustin’s iPhone", KeySteps, 12345},
		{"EXMIN;1,5;hr;2026-11-20T00:00:00+01:00;2026-11-20T23:59:59+01:00;Agustin’s Apple Watch", KeyExercise, 90},
		{"WEIGHT;180;lb;2026-11-20T07:00:00+01:00;2026-11-20T07:00:00+01:00;Health", KeyWeight, 81.6466266},
		{"WAIST;0,84;m;2026-11-20T07:00:00+01:00;2026-11-20T07:00:00+01:00;Health", KeyWaist, 84},
		{"BPSYS;118;mmHg;2026-11-20T07:00:00+01:00;2026-11-20T07:00:00+01:00;Health", KeyBPSys, 118},
		{"HRV;41,7;ms;2026-11-20T03:00:00+01:00;2026-11-20T03:01:00+01:00;Agustin’s Apple Watch", KeyHRV, 41.7},
		{"CR;31;count/min;2026-11-20T18:00:00+01:00;2026-11-20T18:00:00+01:00;Agustin’s Apple Watch", KeyCardio, 31},
		{"KCAL;612,5;kcal;2026-11-20T08:00:00+01:00;2026-11-20T08:00:00+01:00;Cronometer", KeyEnergyIn, 612.5},
		{"KCAL;418,4;kJ;2026-11-20T08:00:00+01:00;2026-11-20T08:00:00+01:00;Cronometer", KeyEnergyIn, 100},
		{"ACTIVE;540;Cal;2026-11-20T00:00:00+01:00;2026-11-20T23:59:00+01:00;Agustin’s Apple Watch", KeyActiveEnergy, 540},
		{"RESTING;1710;kcal;2026-11-20T00:00:00+01:00;2026-11-20T23:59:00+01:00;Agustin’s Apple Watch", KeyRestingEnergy, 1710},
		{"PROTEIN;42,3;g;2026-11-20T08:00:00+01:00;2026-11-20T08:00:00+01:00;Cronometer", KeyProtein, 42.3},
		{"FIBER;6,5;g;2026-11-20T08:00:00+01:00;2026-11-20T08:00:00+01:00;Cronometer", KeyFiber, 6.5},
		{"FAT;1500;mg;2026-11-20T08:00:00+01:00;2026-11-20T08:00:00+01:00;Cronometer", KeyFat, 1.5},
		{"WATER;0,5;L;2026-11-20T08:00:00+01:00;2026-11-20T08:00:00+01:00;Cronometer", KeyWater, 500},
		{"WATER;250;mL;2026-11-20T08:00:00+01:00;2026-11-20T08:00:00+01:00;Cronometer", KeyWater, 250},
	}
	for _, tc := range tests {
		samples, bad := Parse(tc.line)
		if len(bad) != 0 || len(samples) != 1 {
			t.Errorf("%s: samples %v bad %v", tc.line, samples, bad)
			continue
		}
		if samples[0].Key != tc.key || math.Abs(samples[0].Value-tc.want) > 1e-6 {
			t.Errorf("%s: got %s %v, want %s %v", tc.line, samples[0].Key, samples[0].Value, tc.key, tc.want)
		}
	}
}

func TestParseSleepStagesAndMindful(t *testing.T) {
	rows := "SLEEP;Core;hr;2026-11-19T23:10:00+01:00;2026-11-20T01:00:00+01:00;Agustin’s Apple Watch\n" +
		"SLEEP;HKCategoryValueSleepAnalysisAsleepREM;;2026-11-20T01:00:00+01:00;2026-11-20T01:30:00+01:00;Agustin’s Apple Watch\n" +
		"SLEEP;Djup;;2026-11-20T01:30:00+01:00;2026-11-20T02:00:00+01:00;Agustin’s Apple Watch\n" +
		"SLEEP;In Bed;;2026-11-19T23:00:00+01:00;2026-11-20T07:00:00+01:00;Agustin’s iPhone\n" +
		"MIND;0;min;2026-11-20T08:00:00+01:00;2026-11-20T08:05:00+01:00;Agustin’s Apple Watch"
	samples, bad := Parse(rows)
	if len(bad) != 0 {
		t.Fatalf("bad lines: %v", bad)
	}
	want := []string{StageCore, StageREM, StageDeep, StageInBed, ""}
	for i, s := range samples {
		if s.Stage != want[i] {
			t.Errorf("sample %d: stage %q, want %q", i, s.Stage, want[i])
		}
	}
}

func TestParseBadLinesDoNotAbort(t *testing.T) {
	rows := "\n" +
		"RHR;53;count/min;2026-11-18T00:03:14+01:00;2026-11-18T23:54:13+01:00;Watch\r\n" +
		"RHR;53;count/min;2026-11-18T00:03:14+01:00;Watch\n" + // missing field
		"RHR;abc;count/min;2026-11-18T00:03:14+01:00;2026-11-18T23:54:13+01:00;Watch\n" +
		"FOO;1;count;2026-11-18T00:03:14+01:00;2026-11-18T23:54:13+01:00;Watch\n" +
		"HRV;40;s;2026-11-18T00:03:14+01:00;2026-11-18T23:54:13+01:00;Watch\n" +
		"SLEEP;Napping;;2026-11-18T00:03:14+01:00;2026-11-18T23:54:13+01:00;Watch\n" +
		"RHR;53;count/min;2026-11-18 00:03;2026-11-18T23:54:13+01:00;Watch\n" +
		"RHR;53;count/min;2026-11-18T23:54:13+01:00;2026-11-18T00:03:14+01:00;Watch\n"
	samples, bad := Parse(rows)
	if len(samples) != 1 {
		t.Errorf("got %d samples, want 1", len(samples))
	}
	if len(bad) != 7 {
		t.Errorf("got %d bad lines, want 7: %v", len(bad), bad)
	}
}
