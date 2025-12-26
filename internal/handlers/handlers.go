package handlers

import (
	"database/sql"
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
	db        *sql.DB
	templates *template.Template
}

func New(db *sql.DB, templates *template.Template) *Handler {
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
	profile, _ := database.GetUserProfile(h.db)
	date := utils.GetPreviousSundayDate()
	todayHealth, _ := database.GetHealthMetricsByDate(h.db, date)
	todayFitness, _ := database.GetFitnessMetricsByDate(h.db, date)
	todayCognition, _ := database.GetCognitionMetricsByDate(h.db, date)

	data := DashboardData{
		CurrentScore:   currentScore,
		Profile:        profile,
		TodayHealth:    todayHealth,
		TodayFitness:   todayFitness,
		TodayCognition: todayCognition,
	}

	h.templates.ExecuteTemplate(w, "index.html", data)
}

func (h *Handler) HandleSettings(w http.ResponseWriter, r *http.Request) {
	profile, err := database.GetUserProfile(h.db)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// If no profile exists, create an empty/default one
	if profile == nil {
		profile = &models.UserProfile{
			Id:        1,
			BirthDate: "",
			Sex:       "",
			HeightCm:  0,
		}
	}

	data := struct {
		Profile *models.UserProfile
	}{
		Profile: profile,
	}

	err_t := h.templates.ExecuteTemplate(w, "settings.html", data)
	if err_t != nil {
		log.Printf("Template execution error: %v", err_t)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

}

func (h *Handler) HandleRationale(w http.ResponseWriter, r *http.Request) {
	err := h.templates.ExecuteTemplate(w, "rationale.html", nil)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) HandleCurrentScore(w http.ResponseWriter, r *http.Request) {
	currentScore, _ := services.GetCurrentMasterScore(h.db)

	data := struct {
		CurrentScore *models.MasterScore
	}{
		CurrentScore: currentScore,
	}

	h.templates.ExecuteTemplate(w, "score_display", data)
}

func (h *Handler) HandleScores(w http.ResponseWriter, r *http.Request) {
	scores, err := services.GetAllWeeklyScores(h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.templates.ExecuteTemplate(w, "scores.html", scores)
}

func (h *Handler) HandleHealthMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := database.GetRecentHealthMetrics(h.db, 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.templates.ExecuteTemplate(w, "health_metrics.html", metrics)
}

func (h *Handler) HandleFitnessMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := database.GetRecentFitnessMetrics(h.db, 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.templates.ExecuteTemplate(w, "fitness_metrics.html", metrics)
}

func (h *Handler) HandleCognitionMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := database.GetRecentCognitionMetrics(h.db, 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.templates.ExecuteTemplate(w, "cognition_metrics.html", metrics)
}

func (h *Handler) HandleAddHealthMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()

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

	if err := database.SaveHealthMetrics(h.db, health); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refreshScore")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) HandleAddFitnessMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()

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

	if err := database.SaveFitnessMetrics(h.db, fitness); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refreshScore")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) HandleAddCognitionMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()

	var errs []string
	getI := func(key string, label string) int {
		val, err := parseFormInt(r, key)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s is required and must be a number", label))
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

	if err := database.SaveCognitionMetrics(h.db, cognition); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refreshScore")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) HandleCalculateScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.HandleScores(w, r)
}

func (h *Handler) HandleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	existingProfile, err := database.GetUserProfile(h.db)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Error fetching profile", http.StatusInternalServerError)
		return
	}

	var profile models.UserProfile
	if existingProfile != nil {
		profile = *existingProfile
	}

	r.ParseForm()
	profile.BirthDate = r.FormValue("birth_date")
	profile.Sex = r.FormValue("sex")

	height, err := parseFormFloat(r, "height_cm")
	if err != nil {
		http.Error(w, "Invalid height value", http.StatusBadRequest)
		return
	}
	profile.HeightCm = height

	if err := database.SaveUserProfile(h.db, profile); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Profile updated successfully."))
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

	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid decimal number", key)
	}

	return f, nil
}
