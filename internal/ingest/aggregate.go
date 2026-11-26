package ingest

import (
	"sort"
	"time"
)

// Daily metric names, as stored in daily_metrics.metric.
const (
	MetricVO2Max          = "vo2_max"
	MetricRHR             = "rhr"
	MetricHRV             = "hrv_sdnn"
	MetricSteps           = "steps"
	MetricExerciseMin     = "exercise_min"
	MetricWeightKg        = "weight_kg"
	MetricWaistCm         = "waist_cm"
	MetricBPSys           = "bp_sys"
	MetricBPDia           = "bp_dia"
	MetricCardioRecovery  = "cardio_recovery"
	MetricMindfulSessions = "mindful_sessions"
	MetricMindfulMin      = "mindful_min"
	MetricSleepHours      = "sleep_hours"
	MetricSleepBedtime    = "sleep_bedtime"    // minutes after 18:00 the evening before
	MetricSleepAwakeMin   = "sleep_awake_min"  // awake time inside the main sleep period
	MetricSleepAwakenings = "sleep_awakenings" // interruptions inside the main sleep period
	MetricWorkingSets     = "working_sets"
)

// DailyValue is one metric's value for one local calendar date.
type DailyValue struct {
	Date    string // YYYY-MM-DD in the samples' own UTC offset
	Metric  string
	Value   float64
	Samples int
	Source  string
}

// Options tunes Aggregate.
type Options struct {
	// SkipOldestDay drops summed and averaged metrics (steps, exercise, HRV,
	// mindfulness, sleep) for the oldest date in the input. A Shortcut that
	// sends "the last 7 days" may cut that day short, and the full value was
	// already stored by an earlier run. Point metrics (RHR, VO2, weight, …)
	// are kept.
	SkipOldestDay bool
}

type reduction int

const (
	reduceLast reduction = iota
	reduceMean
	reduceMaxSourceSum
)

type metricSpec struct {
	metric string
	reduce reduction
	summed bool // a total or average over the day, so a partial day is wrong
}

var keyMetric = map[string]metricSpec{
	KeyVO2:      {MetricVO2Max, reduceLast, false},
	KeyRHR:      {MetricRHR, reduceLast, false},
	KeyCardio:   {MetricCardioRecovery, reduceLast, false},
	KeyHRV:      {MetricHRV, reduceMean, true},
	KeyWeight:   {MetricWeightKg, reduceMean, false},
	KeyWaist:    {MetricWaistCm, reduceMean, false},
	KeyBPSys:    {MetricBPSys, reduceMean, false},
	KeyBPDia:    {MetricBPDia, reduceMean, false},
	KeySteps:    {MetricSteps, reduceMaxSourceSum, true},
	KeyExercise: {MetricExerciseMin, reduceMaxSourceSum, true},
	KeySets:     {MetricWorkingSets, reduceMaxSourceSum, true},
}

type dayMetric struct{ date, metric string }

type acc struct {
	last    Sample
	sum     float64
	n       int
	sources map[string]float64 // reduceMaxSourceSum: per-source totals; reduceMean: source set
	srcN    map[string]int
}

// Aggregate reduces samples to one value per date and metric:
//   - RHR, VO2 max, cardio recovery: the latest sample of the day
//   - HRV, weight, waist, blood pressure: the mean of the day
//   - steps, exercise minutes, working sets: summed per source, the highest
//     source wins (the Watch and iPhone both count steps)
//   - mindfulness: distinct sessions and their minutes
//   - sleep: see aggregateSleep
//
// The result is sorted by date, then metric.
func Aggregate(samples []Sample, opt Options) []DailyValue {
	oldest := ""
	if opt.SkipOldestDay {
		for _, s := range samples {
			if d := dayOf(s.Start); oldest == "" || d < oldest {
				oldest = d
			}
		}
	}
	skip := func(date string, summed bool) bool {
		return summed && opt.SkipOldestDay && date == oldest
	}

	accs := map[dayMetric]*acc{}
	var sleep []Sample
	mindful := map[string]map[[2]int64]Sample{}

	for _, s := range samples {
		switch s.Key {
		case KeySleep:
			sleep = append(sleep, s)
			continue
		case KeyMindful:
			d := dayOf(s.Start)
			if mindful[d] == nil {
				mindful[d] = map[[2]int64]Sample{}
			}
			mindful[d][[2]int64{s.Start.Unix(), s.End.Unix()}] = s
			continue
		}
		km, ok := keyMetric[s.Key]
		if !ok {
			continue
		}
		dm := dayMetric{dayOf(s.Start), km.metric}
		a := accs[dm]
		if a == nil {
			a = &acc{sources: map[string]float64{}, srcN: map[string]int{}}
			accs[dm] = a
		}
		a.n++
		a.sum += s.Value
		a.sources[s.Source] += s.Value
		a.srcN[s.Source]++
		if a.n == 1 || s.End.After(a.last.End) || (s.End.Equal(a.last.End) && s.Start.After(a.last.Start)) {
			a.last = s
		}
	}

	var out []DailyValue
	for dm, a := range accs {
		km := metricInfo(dm.metric)
		if skip(dm.date, km.summed) {
			continue
		}
		dv := DailyValue{Date: dm.date, Metric: dm.metric}
		switch km.reduce {
		case reduceLast:
			dv.Value, dv.Samples, dv.Source = a.last.Value, a.n, a.last.Source
		case reduceMean:
			dv.Value, dv.Samples, dv.Source = a.sum/float64(a.n), a.n, joinSources(a.sources)
		case reduceMaxSourceSum:
			best := ""
			for src, total := range a.sources {
				if best == "" || total > a.sources[best] || (total == a.sources[best] && src < best) {
					best = src
				}
			}
			dv.Value, dv.Samples, dv.Source = a.sources[best], a.srcN[best], best
		}
		out = append(out, dv)
	}

	for date, sessions := range mindful {
		if skip(date, true) {
			continue
		}
		var minutes float64
		for _, s := range sessions {
			minutes += s.End.Sub(s.Start).Minutes()
		}
		out = append(out,
			DailyValue{Date: date, Metric: MetricMindfulSessions, Value: float64(len(sessions)), Samples: len(sessions)},
			DailyValue{Date: date, Metric: MetricMindfulMin, Value: minutes, Samples: len(sessions)},
		)
	}

	for _, n := range aggregateSleep(sleep) {
		if skip(n.Date, true) {
			continue
		}
		out = append(out, n.dailyValues()...)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date < out[j].Date
		}
		return out[i].Metric < out[j].Metric
	})
	return out
}

func metricInfo(metric string) metricSpec {
	for _, km := range keyMetric {
		if km.metric == metric {
			return km
		}
	}
	return metricSpec{}
}

func joinSources(sources map[string]float64) string {
	if len(sources) == 1 {
		for s := range sources {
			return s
		}
	}
	return "mixed"
}

// dayOf is the calendar date in the time's own offset, which is the offset the
// iPhone recorded it in.
func dayOf(t time.Time) string {
	return t.Format("2006-01-02")
}
