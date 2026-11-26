package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"health-balance/internal/ingest"
	"health-balance/internal/utils"
)

type fakeWriter struct {
	system, user string
	calls        int
	err          error
}

func (f *fakeWriter) Write(_ context.Context, system, user string) (string, string, error) {
	f.calls++
	f.system, f.user = system, user
	if f.err != nil {
		return "", "", f.err
	}
	return "**Sleep** is your main lever.", "fake-model", nil
}

func TestGenerateInsightStoresLatestWeek(t *testing.T) {
	week := utils.GetActiveWeekEndDate()
	values := daily(ingest.MetricRHR, week, 30, 52)
	values = append(values, daily(ingest.MetricSleepHours, week, 14, 7.5)...)
	values = append(values, daily(ingest.MetricSleepBedtime, week, 14, 300)...)
	db := scoreDB(t, values)
	if err := db.SaveUserProfile(scoreProfile); err != nil {
		t.Fatal(err)
	}

	w := &fakeWriter{}
	in, err := GenerateInsight(context.Background(), db, w)
	if err != nil {
		t.Fatal(err)
	}
	if in.WeekEnd != week || in.Model != "fake-model" || !strings.Contains(in.Markdown, "Sleep") {
		t.Errorf("insight %+v", in)
	}
	stored, _ := db.GetLatestInsight()
	if stored == nil || stored.WeekEnd != week {
		t.Errorf("stored %+v", stored)
	}

	for _, want := range []string{
		"years old, male, 180 cm",
		"## Latest week (ending " + week,
		"| Resting HR | marker | measured | 52 bpm |",
		"| Working sets | behaviour | missing |",
		"## Last 12 weeks",
		"| " + week + " | 52 |", // daily table: date, then rhr
		"23:00",                 // bedtime 300 min after 18:00
	} {
		if !strings.Contains(w.user, want) {
			t.Errorf("prompt is missing %q:\n%s", want, w.user)
		}
	}
	if !strings.Contains(w.system, "Biological Reserve Score") {
		t.Error("system prompt missing the score explanation")
	}
}

func TestGenerateInsightErrors(t *testing.T) {
	db := scoreDB(t, nil)
	if _, err := GenerateInsight(context.Background(), db, nil); !errors.Is(err, ErrInsightsDisabled) {
		t.Errorf("no writer: %v", err)
	}
	if _, err := GenerateInsight(context.Background(), db, &fakeWriter{}); !errors.Is(err, ErrProfileRequired) {
		t.Errorf("no profile: %v", err)
	}
	if err := db.SaveUserProfile(scoreProfile); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateInsight(context.Background(), db, &fakeWriter{err: errors.New("overloaded")}); err == nil || !strings.Contains(err.Error(), "overloaded") {
		t.Errorf("writer error: %v", err)
	}
	if in, _ := db.GetLatestInsight(); in != nil {
		t.Errorf("a failed write stored %+v", in)
	}
}

func TestEnsureWeeklyInsightOncePerWeek(t *testing.T) {
	db := scoreDB(t, nil)
	if err := db.SaveUserProfile(scoreProfile); err != nil {
		t.Fatal(err)
	}
	w := &fakeWriter{}
	ensureWeeklyInsight(db, w)
	ensureWeeklyInsight(db, w)
	if w.calls != 1 {
		t.Errorf("wrote %d times, want once for the week", w.calls)
	}
}
