package main

import (
	"html/template"
	"log"
	"net/http"
	"os"

	"health-balance/internal/database"
	"health-balance/internal/handlers"
	"health-balance/internal/middleware"
	"health-balance/internal/services"
)

func staticCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cache for 1 hour (or longer)
		w.Header().Set("Cache-Control", "public, max-age=3600")
		next.ServeHTTP(w, r)
	})
}

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/health.db"
	}

	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatalf("failed to create data directory: %v", err)
	}

	db, err := database.Init(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()

	funcMap := template.FuncMap{
		"divf": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"lt": func(a, b float64) bool { return a < b },
	}

	templates := template.Must(template.New("").Funcs(funcMap).ParseGlob("web/templates/*.html"))

	h := handlers.New(db, templates)

	mux := http.NewServeMux()

	mux.HandleFunc("/", h.HandleHome)
	mux.HandleFunc("/settings", h.HandleSettings)
	mux.HandleFunc("/rationale", h.HandleRationale)
	mux.HandleFunc("/update-profile", h.HandleUpdateProfile)
	mux.HandleFunc("/current-score", h.HandleCurrentScore)
	mux.HandleFunc("/scores", h.HandleScores)
	mux.HandleFunc("/health-metrics", h.HandleHealthMetrics)
	mux.HandleFunc("/add-health-metrics", h.HandleAddHealthMetrics)
	mux.HandleFunc("/delete-health-metric", h.HandleDeleteHealthMetric)
	mux.HandleFunc("/fitness-metrics", h.HandleFitnessMetrics)
	mux.HandleFunc("/add-fitness-metrics", h.HandleAddFitnessMetrics)
	mux.HandleFunc("/delete-fitness-metric", h.HandleDeleteFitnessMetric)
	mux.HandleFunc("/cognition-metrics", h.HandleCognitionMetrics)
	mux.HandleFunc("/add-cognition-metrics", h.HandleAddCognitionMetrics)
	mux.HandleFunc("/delete-cognition-metric", h.HandleDeleteCognitionMetric)
	mux.HandleFunc("/subscribe", h.HandleSubscribe)
	mux.HandleFunc("/unsubscribe", h.HandleUnsubscribe)
	mux.HandleFunc("/ai-summary", h.HandleAiSummary)
	mux.HandleFunc("/health", h.HandleAppHealth)

	mux.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		// Prevent the browser/CDN from caching the service worker script
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		http.ServeFile(w, r, "web/static/sw.js")
	})

	mux.Handle("/static/", staticCache(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static")))))

	services.StartNotificationScheduler(db)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", middleware.RequestLogger(mux)))
}
