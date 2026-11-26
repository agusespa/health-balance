package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"

	"health-balance/internal/database"
	"health-balance/internal/ingest"
	"health-balance/internal/models"
	"health-balance/internal/utils"
)

// DefaultInsightModel writes the weekly insight unless ANTHROPIC_MODEL says otherwise.
const DefaultInsightModel = "claude-opus-5-5"

const (
	insightWeeks     = 12
	insightDailyDays = 14
	insightTimeout   = 5 * time.Minute
)

// ErrInsightsDisabled means neither CLAUDE_RUN_URL nor ANTHROPIC_API_KEY is configured.
var ErrInsightsDisabled = errors.New("insights are disabled: set CLAUDE_RUN_URL or ANTHROPIC_API_KEY")

// InsightWriter turns a prompt into text. Claude implements it; tests fake it.
type InsightWriter interface {
	Write(ctx context.Context, system, user string) (text, model string, err error)
}

// Claude writes insights with the Messages API.
type Claude struct {
	client anthropic.Client
	model  string
}

// NewClaudeFromEnv returns a Claude writer, or nil when ANTHROPIC_API_KEY is unset.
func NewClaudeFromEnv() *Claude {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return nil
	}
	model := os.Getenv("ANTHROPIC_MODEL")
	if model == "" {
		model = DefaultInsightModel
	}
	return &Claude{client: anthropic.NewClient(option.WithAPIKey(key)), model: model}
}

func (c *Claude) Write(ctx context.Context, system, user string) (string, string, error) {
	resp, err := c.client.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:     c.model,
		MaxTokens: 16000,
		System:    []anthropic.BetaTextBlockParam{{Text: system}},
		Messages:  []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(user))},
		OutputConfig: anthropic.BetaOutputConfigParam{
			Effort: anthropic.BetaOutputConfigEffortMedium,
		},
		// If the request is declined by a safety classifier, let the API
		// re-run it on a fallback model instead of returning nothing.
		Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
		Fallbacks: anthropic.BetaFallbacksParamOfDefault(),
	})
	if err != nil {
		return "", "", err
	}
	if resp.StopReason == anthropic.BetaStopReasonRefusal {
		return "", "", fmt.Errorf("the model declined to write the insight")
	}
	var text strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			text.WriteString(t.Text)
		}
	}
	if strings.TrimSpace(text.String()) == "" {
		return "", "", fmt.Errorf("empty response (stop reason %s)", resp.StopReason)
	}
	return text.String(), string(resp.Model), nil
}

var generateMu sync.Mutex

// GenerateInsight writes and stores the insight for the latest completed week.
// Only one generation runs at a time.
func GenerateInsight(ctx context.Context, db database.Querier, w InsightWriter) (*models.Insight, error) {
	if w == nil {
		return nil, ErrInsightsDisabled
	}
	generateMu.Lock()
	defer generateMu.Unlock()

	profile, err := db.GetUserProfile()
	if err != nil {
		return nil, err
	}
	if profile == nil || profile.BirthDate == "" || profile.HeightCm <= 0 {
		return nil, ErrProfileRequired
	}
	scores, err := GetWeeklyScores(db)
	if err != nil {
		return nil, err
	}
	if len(scores) == 0 {
		return nil, errors.New("no weekly scores yet")
	}
	latest := scores[len(scores)-1]
	end, _ := time.Parse(dateLayout, latest.WeekEnd)
	daily, err := db.GetDailyMetrics(end.AddDate(0, 0, -(insightDailyDays-1)).Format(dateLayout), latest.WeekEnd)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, insightTimeout)
	defer cancel()
	text, model, err := w.Write(ctx, insightSystemPrompt, insightUserPrompt(*profile, scores, daily, end))
	if err != nil {
		return nil, fmt.Errorf("write insight: %w", err)
	}
	in := models.Insight{WeekEnd: latest.WeekEnd, Model: model, CreatedAt: time.Now(), Markdown: text}
	if err := db.SaveInsight(in); err != nil {
		return nil, err
	}
	return &in, nil
}

// StartInsightScheduler checks every hour whether the latest completed week
// has an insight and writes one if not; a failed attempt is retried on the
// next check. It does nothing without a writer.
func StartInsightScheduler(db database.Querier, w InsightWriter) {
	if w == nil {
		log.Printf("Insights: neither CLAUDE_RUN_URL nor ANTHROPIC_API_KEY is set, weekly insights are off")
		return
	}
	go func() {
		for {
			ensureWeeklyInsight(db, w)
			time.Sleep(time.Hour)
		}
	}()
}

func ensureWeeklyInsight(db database.Querier, w InsightWriter) {
	week := utils.GetActiveWeekEndDate()
	latest, err := db.GetLatestInsight()
	if err != nil {
		log.Printf("Insights: %v", err)
		return
	}
	if latest != nil && latest.WeekEnd >= week {
		return
	}
	in, err := GenerateInsight(context.Background(), db, w)
	if err != nil {
		log.Printf("Insights: week %s: %v", week, err)
		return
	}
	log.Printf("Insights: wrote week %s with %s", in.WeekEnd, in.Model)
}

const insightSystemPrompt = `You are a longevity and health coach reviewing one person's weekly data. Write for them directly, in English, as "you".

The Biological Reserve Score is a slow-moving estimate of long-term health reserve, updated once per Saturday–Friday week from Apple Health and a gym log; nothing is entered by hand. It started at 1000 in January 2024. Each week an aging tax of (age² / 8000) / 52 of the score is removed, then the score moves 12% of the way toward a target of 1000 plus the points of every metric. Points are capped, and shortfalls cost more than equal gains earn.

Markers (slow measures; when missing, the last value is reused for a while, then fades toward 0 points):
- VO2 max vs the typical value for the person's age and sex
- Waist-to-height ratio vs 0.48
- Resting heart rate vs the person's own 90-day average
- HRV (SDNN) vs the person's own 60-day average
- Blood pressure vs 120/80, measured a few mornings each quarter
- Cardio recovery (heart-rate drop one minute after a workout) vs 25 bpm

Behaviours (scored on a 4-week average):
- Sleep score 0–100 vs 85 (duration up to 50 points with full points from 8 h, bedtime consistency up to 30, interruptions up to 20)
- Steps vs 8,000 a day
- Exercise minutes vs 150 a week
- Gym working sets vs 10 a week
- Mindfulness sessions vs 3 a week

Nutrition is shown but not scored: daily food energy, protein, carbs, fat, fiber and water from a food diary the person fills in by hand, plus active and resting energy burned from the Watch (kcal_out). A day under 1000 kcal eaten is most likely an unfinished diary, not a fast; leave it out. Mention nutrition only when it plausibly explains another metric (weight, sleep, recovery, training) or protein is clearly short for someone who lifts.

Each metric has a status: measured (data this week), carried (last value reused), imputed (fading toward neutral), missing (no data, 0 points). Treat carried, imputed and missing values as gaps in the data, not as facts about this week.

Write:
1. One or two sentences on where the score stands and which way it is heading.
2. **What's helping** and **What's holding you back**: the few metrics that matter most, using the raw values and trends, not just points.
3. **This week**: 3 specific, realistic actions, highest impact first.
4. If data is missing or stale in a way that matters, say which measurement to take.

Keep it under 250 words. Use Markdown with bold headings and flat bullet lists. Be direct and data-driven; do not invent data, and do not give medical diagnoses. If a reading looks clinically concerning, suggest checking with a doctor.`

// metricKcalOut is active plus resting energy, derived for the prompt only.
const metricKcalOut = "kcal_out"

func insightUserPrompt(p models.UserProfile, scores []models.WeeklyScore, daily []models.DailyMetric, weekEnd time.Time) string {
	age, _ := utils.GetAge(&p, weekEnd)
	var b strings.Builder
	fmt.Fprintf(&b, "Person: %d years old, %s, %.0f cm.\n\n", age, p.Sex, p.HeightCm)

	recent := scores
	if len(recent) > insightWeeks {
		recent = recent[len(recent)-insightWeeks:]
	}
	latest := recent[len(recent)-1]

	fmt.Fprintf(&b, "## Latest week (ending %s)\n", latest.WeekEnd)
	fmt.Fprintf(&b, "Score %.1f (%+.1f this week), target %.1f, aging tax %.1f.\n\n", latest.Score, latest.Change, latest.Target, latest.AgingTax)
	b.WriteString("| metric | kind | status | value | reference | points |\n|---|---|---|---|---|---|\n")
	for _, m := range latest.Metrics {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %+.1f |\n", m.Label, m.Kind, m.Status, m.ValueText, m.ReferenceText, m.Points)
	}

	fmt.Fprintf(&b, "\n## Last %d weeks (oldest first): score, then points per metric\n", len(recent))
	b.WriteString("| week | score | change |")
	for _, m := range latest.Metrics {
		b.WriteString(" " + m.Metric + " |")
	}
	b.WriteString("\n|---|---|---|" + strings.Repeat("---|", len(latest.Metrics)) + "\n")
	for _, s := range recent {
		fmt.Fprintf(&b, "| %s | %.1f | %+.1f |", s.WeekEnd, s.Score, s.Change)
		for _, m := range s.Metrics {
			if m.Status == models.StatusMissing {
				b.WriteString(" – |")
			} else {
				fmt.Fprintf(&b, " %+.1f |", m.Points)
			}
		}
		b.WriteString("\n")
	}

	cols := []struct{ metric, label, format string }{
		{ingest.MetricRHR, "rhr", "%.0f"},
		{ingest.MetricHRV, "hrv_ms", "%.0f"},
		{ingest.MetricSleepHours, "sleep_h", "%.1f"},
		{ingest.MetricSleepBedtime, "bedtime", ""},
		{ingest.MetricSteps, "steps", "%.0f"},
		{ingest.MetricExerciseMin, "exercise_min", "%.0f"},
		{ingest.MetricWorkingSets, "sets", "%.0f"},
		{ingest.MetricMindfulSessions, "mindful", "%.0f"},
		{ingest.MetricEnergyIn, "kcal_in", "%.0f"},
		{metricKcalOut, "kcal_out", "%.0f"},
		{ingest.MetricProtein, "protein_g", "%.0f"},
		{ingest.MetricCarbs, "carbs_g", "%.0f"},
		{ingest.MetricFat, "fat_g", "%.0f"},
		{ingest.MetricFiber, "fiber_g", "%.0f"},
		{ingest.MetricWater, "water_ml", "%.0f"},
	}
	byDate := map[string]map[string]float64{}
	for _, d := range daily {
		if byDate[d.Date] == nil {
			byDate[d.Date] = map[string]float64{}
		}
		byDate[d.Date][d.Metric] = d.Value
	}
	for _, day := range byDate {
		active, okA := day[ingest.MetricActiveEnergy]
		resting, okR := day[ingest.MetricRestingEnergy]
		if okA && okR {
			day[metricKcalOut] = active + resting
		}
	}
	dates := make([]string, 0, len(byDate))
	for d := range byDate {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	fmt.Fprintf(&b, "\n## Daily values, last %d days (blank = no data)\n| date |", insightDailyDays)
	for _, c := range cols {
		b.WriteString(" " + c.label + " |")
	}
	b.WriteString("\n|---|" + strings.Repeat("---|", len(cols)) + "\n")
	for _, d := range dates {
		b.WriteString("| " + d + " |")
		for _, c := range cols {
			v, ok := byDate[d][c.metric]
			switch {
			case !ok:
				b.WriteString(" |")
			case c.metric == ingest.MetricSleepBedtime:
				mins := int(v) + 18*60
				fmt.Fprintf(&b, " %02d:%02d |", (mins/60)%24, mins%60)
			default:
				fmt.Fprintf(&b, " "+c.format+" |", v)
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}
