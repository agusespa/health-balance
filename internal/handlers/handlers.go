package handlers

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"health-balance/internal/models"
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
	currentScore, _ := models.GetCurrentMasterScore(h.db)
	recentHealth, _ := models.GetRecentHealthMetrics(h.db, 5)
	recentFitness, _ := models.GetRecentFitnessMetrics(h.db, 5)
	recentCognition, _ := models.GetRecentCognitionMetrics(h.db, 5)
	profile, _ := models.GetUserProfile(h.db)

	date := models.GetPreviousSundayDate()
	todayHealth, _ := models.GetHealthMetricsByDate(h.db, date)
	todayFitness, _ := models.GetFitnessMetricsByDate(h.db, date)
	todayCognition, _ := models.GetCognitionMetricsByDate(h.db, date)

	data := DashboardData{
		CurrentScore:    currentScore,
		RecentHealth:    recentHealth,
		RecentFitness:   recentFitness,
		RecentCognition: recentCognition,
		Profile:         profile,
		TodayHealth:     todayHealth,
		TodayFitness:    todayFitness,
		TodayCognition:  todayCognition,
	}

	h.templates.ExecuteTemplate(w, "index.html", data)
}

func (h *Handler) HandleCurrentScore(w http.ResponseWriter, r *http.Request) {
	currentScore, _ := models.GetCurrentMasterScore(h.db)

	data := struct {
		CurrentScore *models.MasterScore
	}{
		CurrentScore: currentScore,
	}

	h.templates.ExecuteTemplate(w, "score_display", data)
}

func (h *Handler) HandleScores(w http.ResponseWriter, r *http.Request) {
	scores, err := models.GetAllWeeklyScores(h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.templates.ExecuteTemplate(w, "scores.html", scores)
}

func (h *Handler) HandleHealthMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := models.GetRecentHealthMetrics(h.db, 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.templates.ExecuteTemplate(w, "health_metrics.html", metrics)
}

func (h *Handler) HandleFitnessMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := models.GetRecentFitnessMetrics(h.db, 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.templates.ExecuteTemplate(w, "fitness_metrics.html", metrics)
}

func (h *Handler) HandleCognitionMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := models.GetRecentCognitionMetrics(h.db, 5)
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

	health := models.HealthMetrics{
		SleepScore:     parseInt(r.FormValue("sleep_score")),
		WaistCm:        parseFloat(r.FormValue("waist_cm")),
		RHR:            parseInt(r.FormValue("rhr")),
		NutritionScore: parseFloat(r.FormValue("nutrition_score")),
	}

	if err := models.SaveHealthMetrics(h.db, health); err != nil {
		// Return error toast with OOB swap
		toastData := struct {
			Message   string
			IsSuccess bool
		}{
			Message:   "❌ Error saving data",
			IsSuccess: false,
		}
		h.templates.ExecuteTemplate(w, "toast.html", toastData)
		return
	}

	// Get updated metrics
	metrics, _ := models.GetRecentHealthMetrics(h.db, 5)

	// Render metrics table
	var metricsHTML bytes.Buffer
	h.templates.ExecuteTemplate(&metricsHTML, "health_metrics.html", metrics)

	// Render success toast
	toastData := struct {
		Message   string
		IsSuccess bool
	}{
		Message:   "✅ Health data saved!",
		IsSuccess: true,
	}
	var toastHTML bytes.Buffer
	h.templates.ExecuteTemplate(&toastHTML, "toast.html", toastData)

	// Write both responses
	w.Write(metricsHTML.Bytes())
	w.Write(toastHTML.Bytes())
}

func (h *Handler) HandleAddFitnessMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()

	fitness := models.FitnessMetrics{
		VO2Max:         parseFloat(r.FormValue("vo2_max")),
		WeeklyWorkouts: parseInt(r.FormValue("weekly_workouts")),
		DailySteps:     parseInt(r.FormValue("daily_steps")),
		WeeklyMobility: parseInt(r.FormValue("weekly_mobility")),
		CardioRecovery: parseInt(r.FormValue("cardio_recovery")),
	}

	if err := models.SaveFitnessMetrics(h.db, fitness); err != nil {
		// Return error toast with OOB swap
		toastData := struct {
			Message   string
			IsSuccess bool
		}{
			Message:   "❌ Error saving data",
			IsSuccess: false,
		}
		h.templates.ExecuteTemplate(w, "toast.html", toastData)
		return
	}

	// Get updated metrics
	metrics, _ := models.GetRecentFitnessMetrics(h.db, 5)

	// Render metrics table
	var metricsHTML bytes.Buffer
	h.templates.ExecuteTemplate(&metricsHTML, "fitness_metrics.html", metrics)

	// Render success toast
	toastData := struct {
		Message   string
		IsSuccess bool
	}{
		Message:   "✅ Fitness data saved!",
		IsSuccess: true,
	}
	var toastHTML bytes.Buffer
	h.templates.ExecuteTemplate(&toastHTML, "toast.html", toastData)

	// Write both responses
	w.Write(metricsHTML.Bytes())
	w.Write(toastHTML.Bytes())
}

func (h *Handler) HandleAddCognitionMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()

	cognition := models.CognitionMetrics{
		DualNBackLevel:    parseInt(r.FormValue("dual_n_back")),
		ReactionTime:      parseInt(r.FormValue("reaction_time")),
		WeeklyMindfulness: parseInt(r.FormValue("weekly_mindfulness")),
	}

	if err := models.SaveCognitionMetrics(h.db, cognition); err != nil {
		// Return error toast with OOB swap
		toastData := struct {
			Message   string
			IsSuccess bool
		}{
			Message:   "❌ Error saving data",
			IsSuccess: false,
		}
		h.templates.ExecuteTemplate(w, "toast.html", toastData)
		return
	}

	// Get updated metrics
	metrics, _ := models.GetRecentCognitionMetrics(h.db, 5)

	// Render metrics table
	var metricsHTML bytes.Buffer
	h.templates.ExecuteTemplate(&metricsHTML, "cognition_metrics.html", metrics)

	// Render success toast
	toastData := struct {
		Message   string
		IsSuccess bool
	}{
		Message:   "✅ Cognition data saved!",
		IsSuccess: true,
	}
	var toastHTML bytes.Buffer
	h.templates.ExecuteTemplate(&toastHTML, "toast.html", toastData)

	// Write both responses
	w.Write(metricsHTML.Bytes())
	w.Write(toastHTML.Bytes())
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

	// Get existing profile to get the ID
	existingProfile, err := models.GetUserProfile(h.db)
	if err != nil && err != sql.ErrNoRows {
		// Handle potential database errors
		http.Error(w, "Error fetching profile", http.StatusInternalServerError)
		return
	}

	// Create a new profile object or use the existing one
	var profile models.UserProfile
	if existingProfile != nil {
		profile = *existingProfile
	}

	r.ParseForm()
	profile.BirthDate = r.FormValue("birth_date")
	profile.Sex = r.FormValue("sex")
	profile.HeightCm = parseFloat(r.FormValue("height_cm"))

	if err := models.SaveUserProfile(h.db, profile); err != nil {
		// Return error toast with OOB swap
		toastData := struct {
			Message   string
			IsSuccess bool
		}{
			Message:   "❌ Error updating profile",
			IsSuccess: false,
		}
		h.templates.ExecuteTemplate(w, "toast.html", toastData)
		return
	}

	// Return success toast with OOB swap
	toastData := struct {
		Message   string
		IsSuccess bool
	}{
		Message:   "✅ Profile updated!",
		IsSuccess: true,
	}
	h.templates.ExecuteTemplate(w, "toast.html", toastData)
}

func (h *Handler) HandleSettings(w http.ResponseWriter, r *http.Request) {
	profile, err := models.GetUserProfile(h.db)
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

// HandleRationale renders the rationale.html template.
func (h *Handler) HandleRationale(w http.ResponseWriter, r *http.Request) {
	err := h.templates.ExecuteTemplate(w, "rationale.html", nil)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func parseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
