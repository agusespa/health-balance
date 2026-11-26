// Package ingest turns Apple Health samples into one value per day and metric.
//
// The iPhone Shortcut sends lines of the form
//
//	KEY;value;unit;start;end;source
//
// with ISO 8601 dates that carry their UTC offset. Parse reads those lines into
// Samples; Aggregate reduces Samples to DailyValues. The Apple Health export
// backfill builds Samples directly and shares Aggregate.
package ingest

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Shortcut keys.
const (
	KeyVO2      = "VO2"
	KeyRHR      = "RHR"
	KeyHRV      = "HRV"
	KeySleep    = "SLEEP"
	KeySteps    = "STEPS"
	KeyExercise = "EXMIN"
	KeyWeight   = "WEIGHT"
	KeyWaist    = "WAIST"
	KeyBPSys    = "BPSYS"
	KeyBPDia    = "BPDIA"
	KeyCardio   = "CR"
	KeyMindful  = "MIND"
	KeySets     = "SETS" // working sets per day, added by n8n from the gym app

	// Nutrition, written to Health by a food diary app, and the energy
	// burned it is compared with.
	KeyEnergyIn      = "KCAL"
	KeyProtein       = "PROTEIN"
	KeyCarbs         = "CARBS"
	KeyFat           = "FAT"
	KeyFiber         = "FIBER"
	KeyWater         = "WATER"
	KeyActiveEnergy  = "ACTIVE"
	KeyRestingEnergy = "RESTING"
)

// Normalised source names. Any other source keeps its trimmed original name.
const (
	SourceWatch = "watch"
	SourcePhone = "phone"
)

// Sleep stages, stored in Sample.Stage for SLEEP samples.
const (
	StageInBed  = "inbed"
	StageAsleep = "asleep" // unspecified asleep (pre-2022 data)
	StageCore   = "core"
	StageDeep   = "deep"
	StageREM    = "rem"
	StageAwake  = "awake"
)

// Sample is one Health sample with its value converted to the key's canonical unit.
type Sample struct {
	Key    string
	Value  float64 // canonical unit; unused for SLEEP and MIND
	Stage  string  // SLEEP only
	Start  time.Time
	End    time.Time
	Source string
}

// BadLine is an input line that could not be used.
type BadLine struct {
	Line   string `json:"line"`
	Reason string `json:"reason"`
}

// Parse reads newline-separated Shortcut rows. Blank lines are skipped; lines
// that cannot be used are returned in bad and never abort the parse.
func Parse(rows string) (samples []Sample, bad []BadLine) {
	for _, raw := range strings.Split(rows, "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if line == "" {
			continue
		}
		s, err := parseLine(line)
		if err != nil {
			if len(line) > 120 {
				line = line[:120]
			}
			bad = append(bad, BadLine{Line: line, Reason: err.Error()})
			continue
		}
		samples = append(samples, s)
	}
	return samples, bad
}

func parseLine(line string) (Sample, error) {
	f := strings.Split(line, ";")
	if len(f) != 6 {
		return Sample{}, fmt.Errorf("want 6 fields, got %d", len(f))
	}
	return NewSample(f[0], f[1], f[2], f[3], f[4], f[5])
}

// NewSample builds a Sample from the six text fields of a Shortcut row (or
// the matching attributes of an Apple Health export Record), converting the
// value to the key's canonical unit.
func NewSample(key, value, unit, startText, endText, source string) (Sample, error) {
	key = strings.ToUpper(strings.TrimSpace(key))
	value, unit = strings.TrimSpace(value), strings.TrimSpace(unit)

	start, err := parseTime(strings.TrimSpace(startText))
	if err != nil {
		return Sample{}, fmt.Errorf("start: %w", err)
	}
	end, err := parseTime(strings.TrimSpace(endText))
	if err != nil {
		return Sample{}, fmt.Errorf("end: %w", err)
	}
	if end.Before(start) {
		return Sample{}, fmt.Errorf("end before start")
	}

	s := Sample{Key: key, Start: start, End: end, Source: NormalizeSource(source)}

	switch key {
	case KeySleep:
		stage, ok := SleepStage(value)
		if !ok {
			return Sample{}, fmt.Errorf("unknown sleep stage %q", value)
		}
		s.Stage = stage
		return s, nil
	case KeyMindful:
		return s, nil // a session; only its duration matters
	}

	v, err := ParseNumber(value)
	if err != nil {
		return Sample{}, err
	}
	factor, ok := unitFactor(key, unit)
	if !ok {
		return Sample{}, fmt.Errorf("unknown key/unit %s/%q", key, unit)
	}
	s.Value = v * factor
	return s, nil
}

var timeLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04Z07:00",
	"2006-01-02 15:04:05 -0700", // Apple Health export
}

func parseTime(v string) (time.Time, error) {
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("bad time %q", v)
}

// ParseNumber accepts a Swedish decimal comma and space thousands separators
// ("12 345", "48,2").
func ParseNumber(v string) (float64, error) {
	cleaned := strings.Map(func(r rune) rune {
		switch r {
		case ' ', ' ', ' ', '\t':
			return -1
		case ',':
			return '.'
		}
		return r
	}, v)
	n, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, fmt.Errorf("bad number %q", v)
	}
	return n, nil
}

// NormalizeSource maps the Watch and iPhone, under any of their names, to
// SourceWatch and SourcePhone.
func NormalizeSource(v string) string {
	s := strings.TrimSpace(strings.ReplaceAll(v, "’", "'"))
	lower := strings.ToLower(s)
	switch {
	case strings.Contains(lower, "watch"):
		return SourceWatch
	case strings.Contains(lower, "iphone"):
		return SourcePhone
	}
	return s
}

// unitFactor returns the multiplier from unit to the key's canonical unit:
// mL/(kg·min), count/min, ms, count, min, kg, cm, mmHg, kcal, g, mL.
func unitFactor(key, unit string) (float64, bool) {
	u := strings.ToLower(strings.ReplaceAll(unit, " ", ""))
	switch key {
	case KeyVO2:
		if strings.HasPrefix(u, "ml/") {
			return 1, true
		}
	case KeyRHR, KeyCardio:
		switch u {
		case "count/min", "bpm", "slag/min":
			return 1, true
		}
	case KeyHRV:
		if u == "ms" {
			return 1, true
		}
	case KeySteps:
		switch u {
		case "count", "", "steg", "steps":
			return 1, true
		}
	case KeyExercise:
		switch u {
		case "min", "":
			return 1, true
		case "hr", "h":
			return 60, true
		case "s":
			return 1.0 / 60, true
		}
	case KeyWeight:
		switch u {
		case "kg":
			return 1, true
		case "g":
			return 0.001, true
		case "lb":
			return 0.45359237, true
		}
	case KeyWaist:
		switch u {
		case "cm":
			return 1, true
		case "mm":
			return 0.1, true
		case "m":
			return 100, true
		case "in":
			return 2.54, true
		}
	case KeyBPSys, KeyBPDia:
		if u == "mmhg" {
			return 1, true
		}
	case KeySets:
		switch u {
		case "count", "":
			return 1, true
		}
	case KeyEnergyIn, KeyActiveEnergy, KeyRestingEnergy:
		switch u {
		case "kcal", "cal": // Health shows kilocalories as "Cal"
			return 1, true
		case "kj":
			return 1 / 4.184, true
		}
	case KeyProtein, KeyCarbs, KeyFat, KeyFiber:
		switch u {
		case "g":
			return 1, true
		case "mg":
			return 0.001, true
		}
	case KeyWater:
		switch u {
		case "ml":
			return 1, true
		case "cl":
			return 10, true
		case "dl":
			return 100, true
		case "l":
			return 1000, true
		case "floz", "fl_oz_us", "floz_us":
			return 29.5735295625, true
		}
	}
	return 0, false
}

// SleepStage maps the stage text a Shortcut or the export may produce
// (English, Swedish, HealthKit identifiers, HealthKit raw values) to a Stage.
func SleepStage(v string) (string, bool) {
	s := strings.ToLower(strings.TrimSpace(v))
	s = strings.TrimPrefix(s, "hkcategoryvaluesleepanalysis")
	s = strings.NewReplacer(" ", "", "_", "", "-", "").Replace(s)
	switch s {
	case "inbed", "isängen", "0":
		return StageInBed, true
	case "asleep", "asleepunspecified", "sover", "sömn", "1":
		return StageAsleep, true
	case "awake", "vaken", "2":
		return StageAwake, true
	case "core", "asleepcore", "kärna", "kärnsömn", "3":
		return StageCore, true
	case "deep", "asleepdeep", "djup", "djupsömn", "4":
		return StageDeep, true
	case "rem", "asleeprem", "remsömn", "5":
		return StageREM, true
	}
	return "", false
}
