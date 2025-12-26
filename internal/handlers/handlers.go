package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"health-balance/internal/database"
	"health-balance/internal/models"
	"health-balance/internal/services"
	"health-balance/internal/utils"
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
}

func (h *Handler) HandleHome(w http.ResponseWriter, r *http.Request) {
	currentScore, _ := services.GetCurrentMasterScore(h.db)
	profile, _ := h.db.GetUserProfile()
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

	data := struct{ Profile *models.UserProfile }{Profile: profile}
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
	h.render(w, "score_display.html", data)
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
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refreshScore")
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
		WeeklyWorkouts: getI("weekly_workouts"),
		DailySteps:     getI("daily_steps"),
		WeeklyMobility: getI("weekly_mobility"),
		CardioRecovery: getI("cardio_recovery"),
	}

	if len(errs) > 0 {
		http.Error(w, strings.Join(errs, ", "), http.StatusBadRequest)
		return
	}

	if err := h.db.SaveFitnessMetrics(fitness); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refreshScore")
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
		DualNBackLevel:    getI("dual_n_back", "Dual N-Back Level"),
		ReactionTime:      getI("reaction_time", "Reaction Time"),
		WeeklyMindfulness: getI("weekly_mindfulness", "Weekly Mindfulness"),
	}

	if len(errs) > 0 {
		http.Error(w, strings.Join(errs, ", "), http.StatusBadRequest)
		return
	}

	if err := h.db.SaveCognitionMetrics(cognition); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refreshScore")
	w.WriteHeader(http.StatusNoContent)
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
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Profile updated successfully."))
}

// Helpers
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
