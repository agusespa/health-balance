package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"health-balance/internal/database"
	"health-balance/internal/ingest"
	"health-balance/internal/models"
	"health-balance/internal/testutil"
)

func setupTestHandler() (*Handler, *testutil.MockDB) {
	mockDB := &testutil.MockDB{}
	return New(mockDB, nil), mockDB
}

// serve runs a request through the real routes.
func serve(t *testing.T, h *Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return v
}

func realHandler(t *testing.T, profile *models.UserProfile, values []models.DailyMetric) *Handler {
	t.Helper()
	db, err := database.Init(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if profile != nil {
		if err := db.SaveUserProfile(*profile); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.SaveIngest(models.IngestRun{Kind: models.IngestKindShortcut}, values, nil); err != nil {
		t.Fatal(err)
	}
	return New(db, nil)
}

var profile = &models.UserProfile{BirthDate: "1990-06-15", Sex: "male", HeightCm: 180}

func TestDashboardWithoutProfile(t *testing.T) {
	h := realHandler(t, nil, []models.DailyMetric{{Date: "2024-03-01", Metric: ingest.MetricVO2Max, Value: 50}})
	rec := serve(t, h, "GET", "/api/dashboard", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	res := decode[dashboardResponse](t, rec)
	if res.ProfileComplete || res.Score != nil || res.Week.End == "" || res.LatestData != "2024-03-01" {
		t.Errorf("got %+v", res)
	}
}

func TestLatestData(t *testing.T) {
	h := realHandler(t, nil, []models.DailyMetric{{Date: "2024-03-01", Metric: ingest.MetricVO2Max, Value: 50}})
	rec := serve(t, h, "GET", "/api/latest-data", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if got := decode[map[string]string](t, rec)["latest_data"]; got != "2024-03-01" {
		t.Errorf("latest_data %q, want 2024-03-01", got)
	}
}

func TestDashboardScoresAndHistory(t *testing.T) {
	h := realHandler(t, profile, []models.DailyMetric{{Date: "2024-03-01", Metric: ingest.MetricVO2Max, Value: 50}})

	res := decode[dashboardResponse](t, serve(t, h, "GET", "/api/dashboard", ""))
	if !res.ProfileComplete || res.Score == nil || len(res.Score.Metrics) != 11 || res.Score.WeekEnd != res.Week.End {
		t.Fatalf("dashboard %+v", res)
	}

	list := decode[[]scoreSummary](t, serve(t, h, "GET", "/api/scores", ""))
	if len(list) < 100 || list[0].WeekEnd != res.Week.End || list[len(list)-1].WeekEnd != "2024-01-05" {
		t.Errorf("scores: %d weeks, first %s, last %s", len(list), list[0].WeekEnd, list[len(list)-1].WeekEnd)
	}

	week := decode[models.WeeklyScore](t, serve(t, h, "GET", "/api/weeks/2024-03-01", ""))
	for _, m := range week.Metrics {
		if m.Metric == "vo2_max" && (m.Status != models.StatusMeasured || m.Points <= 0) {
			t.Errorf("vo2 in its week: %+v", m)
		}
	}
	if rec := serve(t, h, "GET", "/api/weeks/2024-03-02", ""); rec.Code != http.StatusNotFound {
		t.Errorf("non-Friday week: %d", rec.Code)
	}
	if rec := serve(t, h, "GET", "/api/weeks/nope", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("bad date: %d", rec.Code)
	}
}

func TestScoresNeedProfile(t *testing.T) {
	if rec := serve(t, realHandler(t, nil, nil), "GET", "/api/scores", ""); rec.Code != http.StatusConflict {
		t.Errorf("status %d, want 409", rec.Code)
	}
}

func TestDaily(t *testing.T) {
	h := realHandler(t, nil, []models.DailyMetric{
		{Date: "2026-11-20", Metric: "rhr", Value: 53},
		{Date: "2026-11-20", Metric: "steps", Value: 9000},
		{Date: "2026-11-21", Metric: "rhr", Value: 52},
	})
	rows := decode[[]models.DailyMetric](t, serve(t, h, "GET", "/api/daily?from=2026-11-20&to=2026-11-21&metrics=rhr", ""))
	if len(rows) != 2 || rows[1].Value != 52 {
		t.Errorf("rows %+v", rows)
	}
	empty := serve(t, h, "GET", "/api/daily?from=2020-01-01&to=2020-01-02", "")
	if strings.TrimSpace(empty.Body.String()) != "[]" {
		t.Errorf("empty range: %s", empty.Body)
	}
	for _, q := range []string{"", "?from=2026-11-21&to=2026-11-20", "?from=x&to=2026-11-20"} {
		if rec := serve(t, h, "GET", "/api/daily"+q, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("%q: status %d", q, rec.Code)
		}
	}
}

func TestProfile(t *testing.T) {
	h := realHandler(t, nil, nil)
	if p := decode[models.UserProfile](t, serve(t, h, "GET", "/api/profile", "")); p.BirthDate != "" {
		t.Errorf("empty profile: %+v", p)
	}
	rec := serve(t, h, "PUT", "/api/profile", `{"birth_date":"1990-06-15","sex":"male","height_cm":180}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}
	if p := decode[models.UserProfile](t, serve(t, h, "GET", "/api/profile", "")); p != *profile {
		t.Errorf("profile %+v", p)
	}
	for _, body := range []string{
		`{"birth_date":"2999-01-01","sex":"male","height_cm":180}`,
		`{"birth_date":"1990-06-15","sex":"x","height_cm":180}`,
		`{"birth_date":"1990-06-15","sex":"male","height_cm":20}`,
		`nope`,
	} {
		if rec := serve(t, h, "PUT", "/api/profile", body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d", body, rec.Code)
		}
	}
}

func TestInsights(t *testing.T) {
	h, mock := setupTestHandler()
	if rec := serve(t, h, "GET", "/api/insights", ""); strings.TrimSpace(rec.Body.String()) != "null" {
		t.Errorf("no insight: %s", rec.Body)
	}
	mock.GetLatestInsightFunc = func() (*models.Insight, error) {
		return &models.Insight{WeekEnd: "2026-11-20", Model: "m", Markdown: "**Sleep** is fine <script>alert(1)</script>"}, nil
	}
	res := decode[insightResponse](t, serve(t, h, "GET", "/api/insights", ""))
	if !strings.Contains(res.HTML, "<strong>Sleep</strong>") || strings.Contains(res.HTML, "<script>") {
		t.Errorf("html %q", res.HTML)
	}
	mock.GetLatestInsightFunc = func() (*models.Insight, error) { return nil, errors.New("boom") }
	if rec := serve(t, h, "GET", "/api/insights", ""); rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "boom") {
		t.Errorf("error: %d %s", rec.Code, rec.Body)
	}
}

func TestUnknownAPIRouteIsJSON404(t *testing.T) {
	h, _ := setupTestHandler()
	rec := serve(t, h, "GET", "/api/nope", "")
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "not found") {
		t.Errorf("%d %s", rec.Code, rec.Body)
	}
	if rec := serve(t, h, "GET", "/health", ""); rec.Body.String() != "OK" {
		t.Errorf("health %s", rec.Body)
	}
}

type stubWriter struct{ err error }

func (s stubWriter) Write(context.Context, string, string) (string, string, error) {
	return "## Week\nDo **more** steps.", "stub", s.err
}

func TestRegenerateInsight(t *testing.T) {
	if rec := serve(t, realHandler(t, profile, nil), "POST", "/api/insights", ""); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("no key: %d", rec.Code)
	}

	db, err := database.Init(filepath.Join(t.TempDir(), "insight.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if rec := serve(t, New(db, stubWriter{}), "POST", "/api/insights", ""); rec.Code != http.StatusConflict {
		t.Errorf("no profile: %d", rec.Code)
	}
	if err := db.SaveUserProfile(*profile); err != nil {
		t.Fatal(err)
	}
	if rec := serve(t, New(db, stubWriter{err: errors.New("secret upstream detail")}), "POST", "/api/insights", ""); rec.Code != http.StatusBadGateway || strings.Contains(rec.Body.String(), "secret") {
		t.Errorf("upstream error: %d %s", rec.Code, rec.Body)
	}

	rec := serve(t, New(db, stubWriter{}), "POST", "/api/insights", "")
	res := decode[insightResponse](t, rec)
	if rec.Code != http.StatusOK || res.Model != "stub" || !strings.Contains(res.HTML, "<strong>more</strong>") {
		t.Errorf("%d %+v", rec.Code, res)
	}
	if got := decode[insightResponse](t, serve(t, New(db, nil), "GET", "/api/insights", "")); got.WeekEnd != res.WeekEnd {
		t.Errorf("GET after regenerate: %+v", got)
	}
}
