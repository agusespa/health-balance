package services

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"health-balance/internal/database"
	"health-balance/internal/ingest"
	"health-balance/internal/models"
	"health-balance/internal/utils"
)

// ScoreStartWeek is the first week of score history (a Friday). The score
// starts there at 1000.
const ScoreStartWeek = "2024-01-05"

const (
	dateLayout        = "2006-01-02"
	minDaysPerWeek    = 3 // a daily metric needs this many days to count for a week
	behaviourWindow   = 4 // weeks averaged for behaviours
	behaviourDrift    = 0.5
	markerDrift       = 0.25
	rhrBaselineDays   = 90
	hrvBaselineDays   = 60
	minBaselineDays   = 14
	bedtimeWindow     = 14 // nights used for the usual bedtime
	historyLookbackDy = rhrBaselineDays + bedtimeWindow + 7
	// A good night on the Apple-style 0–100 scale; most of your weeks score
	// 80–97 on it, so this keeps the term from being maxed out every week.
	sleepScoreReference = 85.0
)

// ErrProfileRequired is returned when the profile (birth date, sex, height) is missing.
var ErrProfileRequired = errors.New("profile required for score calculation")

// GetWeeklyScores computes the score for every completed week from
// ScoreStartWeek to the last completed week, from the automatic daily data.
func GetWeeklyScores(db database.Querier) ([]models.WeeklyScore, error) {
	profile, err := db.GetUserProfile()
	if err != nil {
		return nil, err
	}
	if profile == nil || profile.BirthDate == "" || profile.HeightCm <= 0 {
		return nil, ErrProfileRequired
	}
	return ComputeWeeklyScores(db, *profile, ScoreStartWeek, utils.GetActiveWeekEndDate())
}

// ComputeWeeklyScores scores each week whose Friday is between firstWeek and
// lastWeek. The score starts at 1000; each week the aging tax is taken off and
// the score moves 12 % of the way toward the week's target, 1000 plus the sum
// of all metric points.
func ComputeWeeklyScores(db database.Querier, profile models.UserProfile, firstWeek, lastWeek string) ([]models.WeeklyScore, error) {
	first, err := time.Parse(dateLayout, firstWeek)
	if err != nil {
		return nil, err
	}
	last, err := time.Parse(dateLayout, lastWeek)
	if err != nil {
		return nil, err
	}
	if last.Before(first) {
		return nil, nil
	}

	rows, err := db.GetDailyMetrics(first.AddDate(0, 0, -historyLookbackDy).Format(dateLayout), lastWeek)
	if err != nil {
		return nil, fmt.Errorf("load daily metrics: %w", err)
	}
	s := newSeries(rows)
	sleepScores := s.nightlySleepScores()

	tracks := newTracks()
	var out []models.WeeklyScore
	current := defaultMasterScore

	for i, end := 0, first; !end.After(last); i, end = i+1, end.AddDate(0, 0, 7) {
		age, err := utils.GetAge(&profile, end)
		if err != nil {
			return nil, err
		}
		w := week{
			index:   i,
			start:   end.AddDate(0, 0, -6),
			end:     end,
			age:     age,
			profile: profile,
		}
		metrics := tracks.score(w, s, sleepScores)

		target := defaultMasterScore
		for _, m := range metrics {
			target += m.Points
		}
		taxRate := float64(age*age) / 8000.0 / 52.0
		tax := current * taxRate
		post := current - tax
		next := math.Max(0, post+(target-post)*scoreAdjustmentRate)

		out = append(out, models.WeeklyScore{
			WeekEnd:  end.Format(dateLayout),
			Score:    next,
			Change:   next - current,
			AgingTax: tax,
			Target:   target,
			Settle:   settlePoint(target, taxRate),
			Metrics:  metrics,
		})
		current = next
	}
	return out, nil
}

type week struct {
	index      int
	start, end time.Time
	age        int
	profile    models.UserProfile
}

func (w week) from() string { return w.start.Format(dateLayout) }
func (w week) to() string   { return w.end.Format(dateLayout) }

// series holds daily values by metric and date.
type series struct {
	byMetric map[string]map[string]float64
	dates    map[string][]string // sorted dates per metric
}

func newSeries(rows []models.DailyMetric) series {
	s := series{byMetric: map[string]map[string]float64{}, dates: map[string][]string{}}
	for _, r := range rows {
		if s.byMetric[r.Metric] == nil {
			s.byMetric[r.Metric] = map[string]float64{}
		}
		s.byMetric[r.Metric][r.Date] = r.Value
		s.dates[r.Metric] = append(s.dates[r.Metric], r.Date)
	}
	for m := range s.dates {
		sort.Strings(s.dates[m])
	}
	return s
}

// in returns the values with from <= date <= to, in date order.
func (s series) in(metric, from, to string) []float64 {
	dates := s.dates[metric]
	i := sort.SearchStrings(dates, from)
	var out []float64
	for ; i < len(dates) && dates[i] <= to; i++ {
		out = append(out, s.byMetric[metric][dates[i]])
	}
	return out
}

func (s series) mean(metric, from, to string) (float64, int) {
	vals := s.in(metric, from, to)
	if len(vals) == 0 {
		return 0, 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals)), len(vals)
}

func (s series) last(metric, from, to string) (float64, bool) {
	vals := s.in(metric, from, to)
	if len(vals) == 0 {
		return 0, false
	}
	return vals[len(vals)-1], true
}

func (s series) firstDate(metric string) string {
	if d := s.dates[metric]; len(d) > 0 {
		return d[0]
	}
	return ""
}

// nightlySleepScores scores each night against the median bedtime of the
// previous 14 nights.
func (s series) nightlySleepScores() series {
	out := series{byMetric: map[string]map[string]float64{ingest.MetricSleepHours: {}}, dates: map[string][]string{}}
	dates := s.dates[ingest.MetricSleepHours]
	var bedtimes []float64
	for _, d := range dates {
		n := ingest.Night{
			Asleep:     time.Duration(s.byMetric[ingest.MetricSleepHours][d] * float64(time.Hour)),
			BedtimeMin: s.byMetric[ingest.MetricSleepBedtime][d],
			AwakeMin:   s.byMetric[ingest.MetricSleepAwakeMin][d],
			Awakenings: int(s.byMetric[ingest.MetricSleepAwakenings][d]),
		}
		usual := -1.0
		if len(bedtimes) >= 3 {
			usual = median(bedtimes[max(0, len(bedtimes)-bedtimeWindow):])
		}
		out.byMetric[ingest.MetricSleepHours][d] = ingest.SleepScore(n, usual)
		out.dates[ingest.MetricSleepHours] = append(out.dates[ingest.MetricSleepHours], d)
		bedtimes = append(bedtimes, n.BedtimeMin)
	}
	return out
}

func median(vals []float64) float64 {
	sorted := append([]float64(nil), vals...)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// markerTrack carries a marker's points forward: unchanged for carryWeeks,
// then fading toward 0 by markerDrift per week.
type markerTrack struct {
	carryWeeks int
	has        bool
	lastWeek   int
	last       models.MetricScore
}

func (t *markerTrack) step(w week, measured bool, m models.MetricScore) models.MetricScore {
	if measured {
		m.Status = models.StatusMeasured
		t.has, t.lastWeek, t.last = true, w.index, m
		return m
	}
	if !t.has {
		m.Status, m.Points = models.StatusMissing, 0
		m.ValueText = "no data"
		return m
	}
	out := t.last
	since := w.index - t.lastWeek
	out.WeeksOld = since
	if since <= t.carryWeeks {
		out.Status = models.StatusCarried
	} else {
		out.Status = models.StatusImputed
		out.Points = t.last.Points * math.Pow(1-markerDrift, float64(since-t.carryWeeks))
	}
	return out
}

// behaviourTrack averages a habit over the last 4 weeks. A missing week reuses
// the previous value once, then fades it toward neutral by half per week.
type behaviourTrack struct {
	neutral float64
	history []float64
	lastObs float64
	missed  int
}

func (t *behaviourTrack) step(obs float64, ok bool) (smoothed float64, status string) {
	switch {
	case ok:
		t.lastObs, t.missed = obs, 0
		t.history = append(t.history, obs)
		status = models.StatusMeasured
	case len(t.history) == 0:
		return 0, models.StatusMissing
	default:
		t.missed++
		v := t.lastObs
		status = models.StatusCarried
		if t.missed > 1 {
			v = t.neutral + (t.lastObs-t.neutral)*math.Pow(behaviourDrift, float64(t.missed-1))
			status = models.StatusImputed
		}
		t.history = append(t.history, v)
	}
	return averageLastN(t.history, behaviourWindow, func(v float64) float64 { return v }), status
}

// weeksOld is how many weeks ago the habit was last measured, or 0 when it
// was measured in the latest step or never.
func (t *behaviourTrack) weeksOld() int {
	if len(t.history) == 0 {
		return 0
	}
	return t.missed
}

// aged sets a behaviour's WeeksOld from its track.
func aged(m models.MetricScore, t *behaviourTrack) models.MetricScore {
	if m.Status == models.StatusCarried || m.Status == models.StatusImputed {
		m.WeeksOld = t.weeksOld()
	}
	return m
}

type tracks struct {
	vo2, whtr, rhr, hrv, bp, cardio       markerTrack
	sleep, steps, exercise, sets, mindful behaviourTrack
}

func newTracks() *tracks {
	return &tracks{
		vo2:      markerTrack{carryWeeks: 4},
		whtr:     markerTrack{carryWeeks: 8},
		rhr:      markerTrack{carryWeeks: 2},
		hrv:      markerTrack{carryWeeks: 2},
		bp:       markerTrack{carryWeeks: 16},
		cardio:   markerTrack{carryWeeks: 8},
		sleep:    behaviourTrack{neutral: sleepScoreReference},
		steps:    behaviourTrack{neutral: 8000},
		exercise: behaviourTrack{neutral: 150},
		sets:     behaviourTrack{neutral: 10},
		mindful:  behaviourTrack{neutral: 3},
	}
}

func (t *tracks) score(w week, s, sleepScores series) []models.MetricScore {
	from, to := w.from(), w.to()
	var out []models.MetricScore

	// VO2 max vs the age and sex baseline.
	{
		base := models.GetVO2MaxBaseline(w.age, w.profile.Sex)
		v, ok := s.last(ingest.MetricVO2Max, from, to)
		out = append(out, t.vo2.step(w, ok, marker("vo2_max", "VO2 max", v, base,
			fmt1(v), fmt.Sprintf("%s for your age", fmt1(base)), vo2Curve.points(v-base), vo2Curve.maxGain)))
	}
	// Waist-to-height ratio vs 0.48.
	{
		v, ok := s.last(ingest.MetricWaistCm, from, to)
		whtr := v / w.profile.HeightCm
		out = append(out, t.whtr.step(w, ok, marker("whtr", "Waist / height", whtr, 0.48,
			fmt2(whtr), "0.48", whtrCurve.points(0.48-whtr), whtrCurve.maxGain)))
	}
	// Resting heart rate vs your 90-day average.
	{
		v, n := s.mean(ingest.MetricRHR, from, to)
		base, bn := s.mean(ingest.MetricRHR, w.start.AddDate(0, 0, -rhrBaselineDays).Format(dateLayout), w.start.AddDate(0, 0, -1).Format(dateLayout))
		if bn < minBaselineDays {
			base = v
		}
		out = append(out, t.rhr.step(w, n >= minDaysPerWeek, marker("rhr", "Resting HR", v, base,
			fmt0(v)+" bpm", fmt0(base)+" (90-day avg)", rhrCurve.points(base-v), rhrCurve.maxGain)))
	}
	// HRV vs your 60-day average, as a percentage.
	{
		v, n := s.mean(ingest.MetricHRV, from, to)
		base, bn := s.mean(ingest.MetricHRV, w.start.AddDate(0, 0, -hrvBaselineDays).Format(dateLayout), w.start.AddDate(0, 0, -1).Format(dateLayout))
		if bn < minBaselineDays || base <= 0 {
			base = v
		}
		pct := 0.0
		if base > 0 {
			pct = (v/base - 1) * 100
		}
		out = append(out, t.hrv.step(w, n >= minDaysPerWeek, marker("hrv", "HRV", v, base,
			fmt0(v)+" ms", fmt0(base)+" ms (60-day avg)", hrvCurve.points(pct), hrvCurve.maxGain)))
	}
	// Blood pressure vs 120/80, from the mornings measured this week.
	{
		sys, n := s.mean(ingest.MetricBPSys, from, to)
		dia, _ := s.mean(ingest.MetricBPDia, from, to)
		bp := bloodPressurePoints(sys, dia)
		out = append(out, t.bp.step(w, n > 0 && dia > 0, marker("bp", "Blood pressure", sys, 120,
			fmt0(sys)+"/"+fmt0(dia), "120/80", bp, bpSysCurve.maxGain+bpDiaCurve.maxGain)))
	}
	// Cardio recovery vs a 25 bpm drop.
	{
		v, ok := s.last(ingest.MetricCardioRecovery, from, to)
		out = append(out, t.cardio.step(w, ok, marker("cardio_recovery", "Cardio recovery", v, 25,
			fmt0(v)+" bpm", "25 bpm", cardioCurve.points((v-25)/5.0), cardioCurve.maxGain)))
	}

	// Behaviours.
	{
		v, n := sleepScores.mean(ingest.MetricSleepHours, from, to)
		avg, status := t.sleep.step(v, n >= minDaysPerWeek)
		out = append(out, aged(behaviour("sleep_score", "Sleep score", status, avg, sleepScoreReference,
			fmt0(avg), fmt0(sleepScoreReference), sleepCurve.points(avg-sleepScoreReference), sleepCurve.maxGain), &t.sleep))
	}
	{
		v, n := s.mean(ingest.MetricSteps, from, to)
		avg, status := t.steps.step(v, n >= minDaysPerWeek)
		out = append(out, aged(behaviour("steps", "Steps", status, avg, 8000,
			thousands(avg)+"/day", "8,000/day", stepsCurve.points((avg-8000)/2000.0), stepsCurve.maxGain), &t.steps))
	}
	{
		v, n := s.mean(ingest.MetricExerciseMin, from, to)
		avg, status := t.exercise.step(v*7, n >= minDaysPerWeek)
		out = append(out, aged(behaviour("exercise_min", "Exercise", status, avg, 150,
			fmt0(avg)+" min/wk", "150 min/wk", exerciseCurve.points((avg-150)/30.0), exerciseCurve.maxGain), &t.exercise))
	}
	{
		// Gym logging started with the first SETS line; before that it's unknown.
		started := s.firstDate(ingest.MetricWorkingSets)
		vals := s.in(ingest.MetricWorkingSets, from, to)
		var sum float64
		for _, v := range vals {
			sum += v
		}
		avg, status := 0.0, models.StatusMissing
		if started != "" && started <= to {
			avg, status = t.sets.step(sum, len(vals) > 0)
		}
		m := behaviour("working_sets", "Working sets", status, avg, 10,
			fmt0(avg)+"/wk", "10/wk", setsCurve.points((avg-10)/2.0), setsCurve.maxGain)
		if status == models.StatusMissing {
			m.Points, m.ValueText = 0, "no data"
		}
		out = append(out, aged(m, &t.sets))
	}
	{
		// Health only records sessions that happened, so a synced week with
		// none counts as zero.
		_, synced := s.mean(ingest.MetricRHR, from, to)
		if _, stepsDays := s.mean(ingest.MetricSteps, from, to); stepsDays > synced {
			synced = stepsDays
		}
		var sessions float64
		for _, v := range s.in(ingest.MetricMindfulSessions, from, to) {
			sessions += v
		}
		avg, status := t.mindful.step(sessions, synced >= minDaysPerWeek)
		out = append(out, aged(behaviour("mindfulness", "Mindfulness", status, avg, 3,
			fmt1(avg)+"/wk", "3/wk", mindfulCurve.points(avg-3), mindfulCurve.maxGain), &t.mindful))
	}
	return out
}

func marker(metric, label string, value, ref float64, valueText, refText string, points, maxPoints float64) models.MetricScore {
	return models.MetricScore{
		Metric: metric, Label: label, Kind: models.KindMarker,
		Value: round(value, 2), Reference: round(ref, 2),
		ValueText: valueText, ReferenceText: refText, Points: points, MaxPoints: maxPoints,
	}
}

func behaviour(metric, label, status string, value, ref float64, valueText, refText string, points, maxPoints float64) models.MetricScore {
	m := models.MetricScore{
		Metric: metric, Label: label, Kind: models.KindBehaviour, Status: status,
		Value: round(value, 2), Reference: ref,
		ValueText: valueText, ReferenceText: refText, Points: points, MaxPoints: maxPoints,
	}
	if status == models.StatusMissing {
		m.Points, m.ValueText = 0, "no data"
	}
	return m
}

func round(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}

func fmt0(v float64) string { return strconv.FormatFloat(math.Round(v), 'f', 0, 64) }
func fmt1(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) }
func fmt2(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }

func thousands(v float64) string {
	s := fmt0(v)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if neg {
		return "-" + s
	}
	return s
}

const (
	defaultMasterScore  = 1000.0
	scoreAdjustmentRate = 0.12
)

// cappedContribution turns a difference from neutral into points: gains and
// shortfalls have their own slope and cap, so falling short costs more.
func cappedContribution(delta, positiveSlope, negativeSlope, positiveCap, negativeCap float64) float64 {
	if delta >= 0 {
		return math.Min(delta*positiveSlope, positiveCap)
	}
	return math.Max(delta*negativeSlope, -negativeCap)
}

func averageLastN[T any](items []T, window int, value func(T) float64) float64 {
	if len(items) == 0 {
		return 0
	}
	start := max(0, len(items)-window)
	var total float64
	for _, item := range items[start:] {
		total += value(item)
	}
	return total / float64(len(items[start:]))
}

// curve turns a difference from neutral into points with cappedContribution.
type curve struct {
	gainSlope, lossSlope, maxGain, maxLoss float64
}

func (c curve) points(delta float64) float64 {
	return cappedContribution(delta, c.gainSlope, c.lossSlope, c.maxGain, c.maxLoss)
}

var (
	vo2Curve      = curve{2.5, 3.5, 16, 22}
	whtrCurve     = curve{180, 260, 10, 15}
	rhrCurve      = curve{1.0, 1.5, 7, 10}
	hrvCurve      = curve{0.25, 0.3, 5, 8}
	bpSysCurve    = curve{1.0, 1.5, 5, 8}
	bpDiaCurve    = curve{1.0, 1.5, 4, 6}
	cardioCurve   = curve{1.0, 1.5, 4, 6}
	sleepCurve    = curve{0.6, 0.9, 8, 12}
	stepsCurve    = curve{1.0, 1.5, 3, 5}
	exerciseCurve = curve{1.5, 2.5, 6, 9}
	setsCurve     = curve{1.2, 1.5, 6, 8}
	mindfulCurve  = curve{0.6, 1.0, 2, 3}
)

// settlePoint is the score at which a week with this target and tax rate
// changes nothing: the aging tax taken off equals the step toward the target.
func settlePoint(target, taxRate float64) float64 {
	r := scoreAdjustmentRate
	return r * target / (1 - (1-taxRate)*(1-r))
}

// bloodPressurePoints scores systolic against 120 and diastolic against 80.
func bloodPressurePoints(sys, dia float64) float64 {
	if sys <= 0 || dia <= 0 {
		return 0
	}
	return bpSysCurve.points((120-math.Round(sys))/5.0) + bpDiaCurve.points((80-math.Round(dia))/3.0)
}
