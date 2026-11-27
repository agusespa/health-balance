// Package handlers is the JSON API under /api, used by the SvelteKit app and
// the n8n relay.
package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"

	"health-balance/internal/database"
	"health-balance/internal/models"
	"health-balance/internal/services"
	"health-balance/internal/utils"
)

type Handler struct {
	db       database.Querier
	insights services.InsightWriter // nil when no insight writer is configured
}

func New(db database.Querier, insights services.InsightWriter) *Handler {
	return &Handler{db: db, insights: insights}
}

// Routes registers every API route on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.HandleAppHealth)
	mux.HandleFunc("GET /api/dashboard", h.HandleDashboard)
	mux.HandleFunc("GET /api/latest-data", h.HandleLatestData)
	mux.HandleFunc("GET /api/scores", h.HandleScores)
	mux.HandleFunc("GET /api/drivers", h.HandleDrivers)
	mux.HandleFunc("GET /api/weeks/{date}", h.HandleWeek)
	mux.HandleFunc("GET /api/daily", h.HandleDaily)
	mux.HandleFunc("GET /api/profile", h.HandleGetProfile)
	mux.HandleFunc("PUT /api/profile", h.HandlePutProfile)
	mux.HandleFunc("GET /api/insights", h.HandleInsights)
	mux.HandleFunc("POST /api/insights", h.HandleRegenerateInsight)
	mux.HandleFunc("POST /api/ingest/health", h.HandleIngestHealth)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, apiError{"not found"})
	})
}

type apiError struct {
	Error string `json:"error"`
}

func (h *Handler) HandleAppHealth(w http.ResponseWriter, r *http.Request) {
	if _, err := fmt.Fprint(w, "OK"); err != nil {
		log.Printf("Failed to write health check response: %v", err)
	}
}

type weekInfo struct {
	End   string `json:"end"`
	Range string `json:"range"`
}

type dashboardResponse struct {
	ProfileComplete bool                       `json:"profile_complete"`
	Week            weekInfo                   `json:"week"`
	Score           *models.WeeklyScore        `json:"score"`
	LatestData      string                     `json:"latest_data"`
	Nutrition       *services.NutritionSummary `json:"nutrition"`
}

// HandleDashboard returns the latest completed week's score with its
// per-metric breakdown, the newest day with any data, and nutrition over the
// last full days (today is still being logged).
func (h *Handler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	res := dashboardResponse{Week: weekInfo{End: utils.GetActiveWeekEndDate(), Range: utils.GetActiveWeekDateRange()}}

	latest, err := h.db.GetLatestDataDate()
	if err != nil {
		internalError(w, "latest data", err)
		return
	}
	res.LatestData = latest

	nutrition, err := services.GetNutritionSummary(h.db, time.Now().AddDate(0, 0, -1))
	if err != nil {
		internalError(w, "nutrition", err)
		return
	}
	res.Nutrition = nutrition

	scores, err := services.GetWeeklyScores(h.db)
	switch {
	case errors.Is(err, services.ErrProfileRequired):
	case err != nil:
		internalError(w, "scores", err)
		return
	default:
		res.ProfileComplete = true
		if len(scores) > 0 {
			res.Score = &scores[len(scores)-1]
		}
	}
	writeJSON(w, http.StatusOK, res)
}

// HandleLatestData returns the newest day with any stored value, or "" when
// there is none, for pages that only need to know whether data is arriving.
func (h *Handler) HandleLatestData(w http.ResponseWriter, r *http.Request) {
	latest, err := h.db.GetLatestDataDate()
	if err != nil {
		internalError(w, "latest data", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"latest_data": latest})
}

type scoreSummary struct {
	WeekEnd  string  `json:"week_end"`
	Score    float64 `json:"score"`
	Change   float64 `json:"change"`
	Target   float64 `json:"target"`
	AgingTax float64 `json:"aging_tax"`
}

// HandleScores lists every week's score, newest first, without the breakdown.
func (h *Handler) HandleScores(w http.ResponseWriter, r *http.Request) {
	scores, ok := h.weeklyScores(w)
	if !ok {
		return
	}
	out := make([]scoreSummary, 0, len(scores))
	for i := len(scores) - 1; i >= 0; i-- {
		s := scores[i]
		out = append(out, scoreSummary{s.WeekEnd, s.Score, s.Change, s.Target, s.AgingTax})
	}
	writeJSON(w, http.StatusOK, out)
}

type driverWeek struct {
	WeekEnd string    `json:"week_end"`
	Score   float64   `json:"score"`
	Points  []float64 `json:"points"`
	// Aging is the week's aging tax in target points, to set against Points.
	Aging float64 `json:"aging"`
}

type driversResponse struct {
	Metrics []models.MetricScore `json:"metrics"`
	Weeks   []driverWeek         `json:"weeks"`
}

// HandleDrivers returns every week's score with each metric's points, oldest
// first. Metrics is the latest week's breakdown and fixes the order of each
// week's Points.
func (h *Handler) HandleDrivers(w http.ResponseWriter, r *http.Request) {
	scores, ok := h.weeklyScores(w)
	if !ok {
		return
	}
	res := driversResponse{Metrics: []models.MetricScore{}, Weeks: make([]driverWeek, 0, len(scores))}
	if len(scores) > 0 {
		res.Metrics = scores[len(scores)-1].Metrics
	}
	for _, s := range scores {
		byMetric := make(map[string]float64, len(s.Metrics))
		for _, m := range s.Metrics {
			byMetric[m.Metric] = m.Points
		}
		points := make([]float64, len(res.Metrics))
		for i, m := range res.Metrics {
			points[i] = byMetric[m.Metric]
		}
		res.Weeks = append(res.Weeks, driverWeek{s.WeekEnd, s.Score, points, services.AgingPoints(s.AgingTax)})
	}
	writeJSON(w, http.StatusOK, res)
}

// HandleWeek returns one week's score with its per-metric breakdown.
func (h *Handler) HandleWeek(w http.ResponseWriter, r *http.Request) {
	date := r.PathValue("date")
	if !validDate(date) {
		writeJSON(w, http.StatusBadRequest, apiError{"date must be YYYY-MM-DD"})
		return
	}
	scores, ok := h.weeklyScores(w)
	if !ok {
		return
	}
	for _, s := range scores {
		if s.WeekEnd == date {
			writeJSON(w, http.StatusOK, s)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, apiError{"no score for that week"})
}

func (h *Handler) weeklyScores(w http.ResponseWriter) ([]models.WeeklyScore, bool) {
	scores, err := services.GetWeeklyScores(h.db)
	if errors.Is(err, services.ErrProfileRequired) {
		writeJSON(w, http.StatusConflict, apiError{"set your birth date, sex and height in Settings first"})
		return nil, false
	}
	if err != nil {
		internalError(w, "scores", err)
		return nil, false
	}
	return scores, true
}

// HandleDaily returns daily values between from and to (inclusive, at most
// about six years), optionally limited to a comma-separated list of metrics.
func (h *Handler) HandleDaily(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to := q.Get("from"), q.Get("to")
	if !validDate(from) || !validDate(to) || from > to {
		writeJSON(w, http.StatusBadRequest, apiError{"from and to must be YYYY-MM-DD, from <= to"})
		return
	}
	var metrics []string
	if m := q.Get("metrics"); m != "" {
		metrics = strings.Split(m, ",")
	}
	rows, err := h.db.GetDailyMetrics(from, to, metrics...)
	if err != nil {
		internalError(w, "daily metrics", err)
		return
	}
	if rows == nil {
		rows = []models.DailyMetric{}
	}
	writeJSON(w, http.StatusOK, rows)
}

func (h *Handler) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := h.db.GetUserProfile()
	if err != nil {
		internalError(w, "profile", err)
		return
	}
	if profile == nil {
		profile = &models.UserProfile{}
	}
	writeJSON(w, http.StatusOK, profile)
}

func (h *Handler) HandlePutProfile(w http.ResponseWriter, r *http.Request) {
	var p models.UserProfile
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{"invalid JSON body"})
		return
	}
	var problems []string
	if birth, err := time.Parse("2006-01-02", p.BirthDate); err != nil || birth.After(time.Now()) {
		problems = append(problems, "birth date must be a past date (YYYY-MM-DD)")
	}
	if p.Sex != "male" && p.Sex != "female" {
		problems = append(problems, "sex must be male or female")
	}
	if p.HeightCm < 100 || p.HeightCm > 250 {
		problems = append(problems, "height must be between 100 and 250 cm")
	}
	if len(problems) > 0 {
		writeJSON(w, http.StatusBadRequest, apiError{strings.Join(problems, "; ")})
		return
	}
	if err := h.db.SaveUserProfile(p); err != nil {
		internalError(w, "save profile", err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

type insightResponse struct {
	WeekEnd   string    `json:"week_end"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	HTML      string    `json:"html"`
}

// HandleInsights returns the latest cached insight as sanitised HTML, or null.
func (h *Handler) HandleInsights(w http.ResponseWriter, r *http.Request) {
	in, err := h.db.GetLatestInsight()
	if err != nil {
		internalError(w, "insight", err)
		return
	}
	if in == nil {
		writeJSON(w, http.StatusOK, nil)
		return
	}
	writeInsight(w, in)
}

// HandleRegenerateInsight writes a fresh insight for the latest completed
// week now. It can take a minute.
func (h *Handler) HandleRegenerateInsight(w http.ResponseWriter, r *http.Request) {
	in, err := services.GenerateInsight(r.Context(), h.db, h.insights)
	switch {
	case errors.Is(err, services.ErrInsightsDisabled):
		writeJSON(w, http.StatusServiceUnavailable, apiError{"insights are off: set CLAUDE_RUN_URL or ANTHROPIC_API_KEY on the server"})
	case errors.Is(err, services.ErrProfileRequired):
		writeJSON(w, http.StatusConflict, apiError{"set your birth date, sex and height in Settings first"})
	case err != nil:
		log.Printf("regenerate insight: %v", err)
		writeJSON(w, http.StatusBadGateway, apiError{"couldn't write insights right now; try again later"})
	default:
		writeInsight(w, in)
	}
}

func writeInsight(w http.ResponseWriter, in *models.Insight) {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(in.Markdown), &buf); err != nil {
		internalError(w, "render insight", err)
		return
	}
	writeJSON(w, http.StatusOK, insightResponse{
		WeekEnd:   in.WeekEnd,
		Model:     in.Model,
		CreatedAt: in.CreatedAt,
		HTML:      string(bluemonday.UGCPolicy().SanitizeBytes(buf.Bytes())),
	})
}

func validDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

func internalError(w http.ResponseWriter, what string, err error) {
	log.Printf("%s: %v", what, err)
	writeJSON(w, http.StatusInternalServerError, apiError{"internal error"})
}
