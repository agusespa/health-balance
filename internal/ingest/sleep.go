package ingest

import (
	"math"
	"sort"
	"time"
)

const (
	// A night belongs to the date you wake up on: samples starting from 18:00
	// the evening before until 18:00 that day.
	nightCutoffHour = 18
	// Asleep segments closer than this are one stretch of sleep.
	awakeningMinGap = time.Minute
	// Asleep segments further apart than this are separate sleep periods
	// (for example a nap); only the longest period of the night counts.
	sleepPeriodGap = 2 * time.Hour
)

// Night is the main sleep period that ends on Date.
type Night struct {
	Date       string
	Source     string
	Asleep     time.Duration
	BedtimeMin float64 // first asleep minute, counted from 18:00 the evening before
	AwakeMin   float64 // awake time between falling asleep and the final wake-up
	Awakenings int
	Segments   int
}

func (n Night) dailyValues() []DailyValue {
	return []DailyValue{
		{Date: n.Date, Metric: MetricSleepHours, Value: n.Asleep.Hours(), Samples: n.Segments, Source: n.Source},
		{Date: n.Date, Metric: MetricSleepBedtime, Value: n.BedtimeMin, Samples: n.Segments, Source: n.Source},
		{Date: n.Date, Metric: MetricSleepAwakeMin, Value: n.AwakeMin, Samples: n.Segments, Source: n.Source},
		{Date: n.Date, Metric: MetricSleepAwakenings, Value: float64(n.Awakenings), Samples: n.Segments, Source: n.Source},
	}
}

type interval struct{ start, end time.Time }

// aggregateSleep builds one Night per wake-up date from sleep-stage samples,
// merging every source's segments. Core, deep, REM and unspecified asleep
// count as asleep. Awake time is the gaps between asleep segments, so it works
// for stage data and for pre-2022 data without stages.
func aggregateSleep(samples []Sample) []Night {
	ivs := map[string][]interval{} // night date → asleep intervals
	sources := map[string]map[string]bool{}
	for _, s := range samples {
		if !Asleep(s.Stage) {
			continue
		}
		d := NightOf(s.Start)
		ivs[d] = append(ivs[d], interval{s.Start, s.End})
		if sources[d] == nil {
			sources[d] = map[string]bool{}
		}
		sources[d][s.Source] = true
	}

	var nights []Night
	for date, list := range ivs {
		if n, ok := buildNight(date, joinSources(sources[date]), list); ok {
			nights = append(nights, n)
		}
	}
	sort.Slice(nights, func(i, j int) bool { return nights[i].Date < nights[j].Date })
	return nights
}

// Asleep reports whether a sleep stage counts as asleep.
func Asleep(stage string) bool {
	switch stage {
	case StageAsleep, StageCore, StageDeep, StageREM:
		return true
	}
	return false
}

func buildNight(date, source string, ivs []interval) (Night, bool) {
	merged := mergeIntervals(ivs, awakeningMinGap)
	if len(merged) == 0 {
		return Night{}, false
	}

	// Split into sleep periods and keep the one with the most asleep time.
	var periods [][]interval
	for i, iv := range merged {
		if i == 0 || iv.start.Sub(merged[i-1].end) > sleepPeriodGap {
			periods = append(periods, nil)
		}
		periods[len(periods)-1] = append(periods[len(periods)-1], iv)
	}
	var main []interval
	var mainAsleep time.Duration
	for _, p := range periods {
		if a := totalDuration(p); a > mainAsleep {
			main, mainAsleep = p, a
		}
	}
	if mainAsleep <= 0 {
		return Night{}, false
	}

	first, last := main[0].start, main[len(main)-1].end
	y, m, d := first.Date()
	if first.Hour() < nightCutoffHour {
		y, m, d = first.AddDate(0, 0, -1).Date()
	}
	evening := time.Date(y, m, d, nightCutoffHour, 0, 0, 0, first.Location())

	return Night{
		Date:       date,
		Source:     source,
		Asleep:     mainAsleep,
		BedtimeMin: first.Sub(evening).Minutes(),
		AwakeMin:   (last.Sub(first) - mainAsleep).Minutes(),
		Awakenings: len(main) - 1,
		Segments:   len(ivs),
	}, true
}

// mergeIntervals sorts intervals and joins those that overlap or are closer
// than gap.
func mergeIntervals(ivs []interval, gap time.Duration) []interval {
	sorted := append([]interval(nil), ivs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].start.Before(sorted[j].start) })
	var out []interval
	for _, iv := range sorted {
		if !iv.end.After(iv.start) {
			continue
		}
		if n := len(out); n > 0 && iv.start.Sub(out[n-1].end) < gap {
			if iv.end.After(out[n-1].end) {
				out[n-1].end = iv.end
			}
			continue
		}
		out = append(out, iv)
	}
	return out
}

func totalDuration(ivs []interval) time.Duration {
	var d time.Duration
	for _, iv := range ivs {
		d += iv.end.Sub(iv.start)
	}
	return d
}

// NightOf is the wake-up date a sleep sample belongs to.
func NightOf(t time.Time) string {
	if t.Hour() >= nightCutoffHour {
		return dayOf(t.AddDate(0, 0, 1))
	}
	return dayOf(t)
}

// SleepScore approximates the Health app's 0–100 sleep score from the three
// parts Apple describes:
//   - duration, up to 50: full from 8 h asleep, zero at 4 h
//   - bedtime, up to 30: full within 15 min of your usual bedtime, zero at 75 min off
//   - interruptions, up to 20: full with up to 4 awakenings and 20 awake
//     minutes (newer watchOS logs many brief awakenings, so it is lenient)
//
// usualBedtimeMin is your typical BedtimeMin (for example the median of the
// previous 14 nights); pass a negative value when there is no history, which
// gives full bedtime points.
func SleepScore(n Night, usualBedtimeMin float64) float64 {
	hours := n.Asleep.Hours()
	duration := 50 * clamp01((hours-4)/(8-4))

	bedtime := 30.0
	if usualBedtimeMin >= 0 {
		off := math.Abs(n.BedtimeMin - usualBedtimeMin)
		bedtime = 30 * clamp01(1-(off-15)/(75-15))
	}

	penalty := 2*math.Max(0, float64(n.Awakenings-4)) + 0.3*math.Max(0, n.AwakeMin-20)
	interruptions := math.Max(0, 20-penalty)

	return math.Round((duration+bedtime+interruptions)*10) / 10
}

func clamp01(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}
