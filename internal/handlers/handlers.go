package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"encoding/json"
	"health-balance/internal/database"
	"health-balance/internal/models"
	"health-balance/internal/services"
	"health-balance/internal/utils"
	"os"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
)

type Handler struct {
	db        database.Querier
	templates *template.Template
}

func New(db database.Querier, templates *template.Template) *Handler {
	return &Handler{
		db:        db,
		templates: templates,
	}
}

type DashboardData struct {
	CurrentScore    *models.MasterScore
	RecentHealth    []models.HealthMetrics
	RecentFitness   []models.FitnessMetrics
	RecentCognition []models.CognitionMetrics
	Profile         *models.UserProfile
	TodayHealth     *models.HealthMetrics
	TodayFitness    *models.FitnessMetrics
	TodayCognition  *models.CognitionMetrics
	HasProfile      bool
}

func (h *Handler) HandleHome(w http.ResponseWriter, r *http.Request) {
	currentScore, _ := services.GetCurrentMasterScore(h.db)
	profile, _ := h.db.GetUserProfile()
	hasProfile := profile != nil && profile.BirthDate != "" && profile.Sex != "" && profile.HeightCm > 0

	date := utils.GetPreviousSundayDate()

	todayHealth, _ := h.db.GetHealthMetricsByDate(date)
	todayFitness, _ := h.db.GetFitnessMetricsByDate(date)
	todayCognition, _ := h.db.GetCognitionMetricsByDate(date)

	data := DashboardData{
		CurrentScore:   currentScore,
		Profile:        profile,
		TodayHealth:    todayHealth,
		TodayFitness:   todayFitness,
		TodayCognition: todayCognition,
		HasProfile:     hasProfile,
	}

	h.render(w, "index.html", data)
}

func (h *Handler) HandleSettings(w http.ResponseWriter, r *http.Request) {
	profile, err := h.db.GetUserProfile()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if profile == nil {
		profile = &models.UserProfile{
			BirthDate: "",
			Sex:       "",
			HeightCm:  0,
		}
	}

	sub, _ := h.db.GetAnyPushSubscription()

	data := struct {
		Profile        *models.UserProfile
		Subscription   *models.PushSubscription
		VapidPublicKey string
	}{
		Profile:        profile,
		Subscription:   sub,
		VapidPublicKey: os.Getenv("VAPID_PUBLIC_KEY"),
	}
	err_t := h.templates.ExecuteTemplate(w, "settings.html", data)
	if err_t != nil {
		log.Printf("Template execution error: %v", err_t)
	}
}

func (h *Handler) HandleRationale(w http.ResponseWriter, r *http.Request) {
	h.render(w, "rationale.html", nil)
}

func (h *Handler) HandleCurrentScore(w http.ResponseWriter, r *http.Request) {
	currentScore, _ := services.GetCurrentMasterScore(h.db)
	data := struct{ CurrentScore *models.MasterScore }{CurrentScore: currentScore}
	h.render(w, "score_display", data)
}

func (h *Handler) HandleScores(w http.ResponseWriter, r *http.Request) {
	scores, err := services.GetAllWeeklyScores(h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.render(w, "scores.html", scores)
}

func (h *Handler) HandleHealthMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.db.GetRecentHealthMetrics(5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.render(w, "health_metrics.html", metrics)
}

func (h *Handler) HandleFitnessMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.db.GetRecentFitnessMetrics(5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.render(w, "fitness_metrics.html", metrics)
}

func (h *Handler) HandleCognitionMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.db.GetRecentCognitionMetrics(5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.render(w, "cognition_metrics.html", metrics)
}

func (h *Handler) HandleAddHealthMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	var errs []string
	getF := func(key string) float64 {
		val, err := parseFormFloat(r, key)
		if err != nil {
			errs = append(errs, err.Error())
		}
		return val
	}
	getI := func(key string) int {
		val, err := parseFormInt(r, key)
		if err != nil {
			errs = append(errs, err.Error())
		}
		return val
	}

	health := models.HealthMetrics{
		SleepScore:     getI("sleep_score"),
		WaistCm:        getF("waist_cm"),
		RHR:            getI("rhr"),
		NutritionScore: getF("nutrition_score"),
	}

	if len(errs) > 0 {
		http.Error(w, strings.Join(errs, ", "), http.StatusBadRequest)
		return
	}

	if err := h.db.SaveHealthMetrics(health); err != nil {
		log.Printf("Error saving health metrics: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"refreshScore":null, "showToast":"Health data saved successfully"}`)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) HandleAddFitnessMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	var errs []string
	getI := func(key string) int {
		val, err := parseFormInt(r, key)
		if err != nil {
			errs = append(errs, err.Error())
		}
		return val
	}
	getF := func(key string) float64 {
		val, err := parseFormFloat(r, key)
		if err != nil {
			errs = append(errs, err.Error())
		}
		return val
	}

	fitness := models.FitnessMetrics{
		VO2Max:         getF("vo2_max"),
		Workouts:       getI("workouts"),
		DailySteps:     getI("daily_steps"),
		Mobility:       getI("mobility"),
		CardioRecovery: getI("cardio_recovery"),
	}

	if len(errs) > 0 {
		http.Error(w, strings.Join(errs, ", "), http.StatusBadRequest)
		return
	}

	if err := h.db.SaveFitnessMetrics(fitness); err != nil {
		log.Printf("Error saving fitness metrics: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"refreshScore":null, "showToast":"Fitness data saved successfully"}`)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) HandleAddCognitionMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	var errs []string
	getI := func(key string, label string) int {
		val, err := parseFormInt(r, key)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s is required", label))
		}
		return val
	}

	cognition := models.CognitionMetrics{
		Mindfulness:    getI("mindfulness", "Mindfulness"),
		DeepLearning:   getI("deep_learning", "Deep Learning"),
		DualNBackLevel: getI("dual_n_back", "Dual N-Back Level"),
		ReactionTime:   getI("reaction_time", "Reaction Time"),
	}

	if len(errs) > 0 {
		http.Error(w, strings.Join(errs, ", "), http.StatusBadRequest)
		return
	}

	if err := h.db.SaveCognitionMetrics(cognition); err != nil {
		log.Printf("Error saving cognition metrics: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"refreshScore":null, "showToast":"Cognition data saved successfully"}`)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) HandleDeleteHealthMetric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		http.Error(w, "Date is required", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteHealthMetrics(date); err != nil {
		log.Printf("Error deleting health metric: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"refreshScore":null, "showToast":"Entry deleted successfully"}`)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HandleDeleteFitnessMetric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		http.Error(w, "Date is required", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteFitnessMetrics(date); err != nil {
		log.Printf("Error deleting fitness metric: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"refreshScore":null, "showToast":"Entry deleted successfully"}`)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HandleDeleteCognitionMetric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		http.Error(w, "Date is required", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteCognitionMetrics(date); err != nil {
		log.Printf("Error deleting cognition metric: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"refreshScore":null, "showToast":"Entry deleted successfully"}`)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HandleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	existingProfile, _ := h.db.GetUserProfile()
	var profile models.UserProfile
	if existingProfile != nil {
		profile = *existingProfile
	}

	profile.BirthDate = r.FormValue("birth_date")
	profile.Sex = r.FormValue("sex")
	height, _ := parseFormFloat(r, "height_cm")
	profile.HeightCm = height

	if err := h.db.SaveUserProfile(profile); err != nil {
		log.Printf("Error saving user profile: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"showToast":"Biomarkers updated successfully"}`)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.PushSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding subscription: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	sub := req.Subscription
	sub.ReminderDay = req.ReminderDay
	sub.ReminderTime = req.ReminderTime
	sub.Timezone = req.Timezone

	if err := h.db.SavePushSubscription(sub); err != nil {
		log.Printf("Error saving subscription: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "success"}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func (h *Handler) HandleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding unsubscribe: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Endpoint == "" {
		// Fallback: delete the most recent subscription if no endpoint provided
		sub, err := h.db.GetAnyPushSubscription()
		if err != nil {
			log.Printf("Error getting any subscription for unsubscribe: %v", err)
		} else if sub != nil {
			req.Endpoint = sub.Endpoint
		}
	}

	if req.Endpoint != "" {
		if err := h.db.DeletePushSubscription(req.Endpoint); err != nil {
			log.Printf("Error deleting subscription: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "success"}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func (h *Handler) HandleAppHealth(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprint(w, "OK")
	if err != nil {
		log.Printf("Failed to write health check response: %v", err)
	}
}

func (h *Handler) HandleAiSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := services.GetHealthSummary(h.db)
	if err != nil {
		log.Printf("AI summary error: %v", err)
		if _, err := fmt.Fprintf(w, `<div class="text-red-600 p-4 bg-red-50 rounded">Failed to generate AI summary. Please ensure API_KEY is set.</div>`); err != nil {
			log.Printf("Error writing error response: %v", err)
		}
		return
	}

	// Convert markdown to HTML
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(summary), &buf); err != nil {
		log.Printf("Markdown conversion error: %v", err)
		if _, err := fmt.Fprintf(w, `<div class="text-red-600 p-4 bg-red-50 rounded">Failed to render summary.</div>`); err != nil {
			log.Printf("Error writing error response: %v", err)
		}
		return
	}

	// Sanitize HTML to only allow safe tags
	policy := bluemonday.UGCPolicy()
	safeHTML := policy.SanitizeBytes(buf.Bytes())

	if _, err := fmt.Fprintf(w, `<div>%s</div>`, safeHTML); err != nil {
		log.Printf("Error writing AI summary response: %v", err)
	}
}

func parseFormInt(r *http.Request, key string) (int, error) {
	val := r.FormValue(key)
	if val == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	return strconv.Atoi(val)
}

func parseFormFloat(r *http.Request, key string) (float64, error) {
	val := r.FormValue(key)
	if val == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	return strconv.ParseFloat(val, 64)
}

func (h *Handler) render(w http.ResponseWriter, name string, data any) {
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Render error [%s]: %v", name, err)
		http.Error(w, "An unexpected error occurred", http.StatusInternalServerError)
	}
}
