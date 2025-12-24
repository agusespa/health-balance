package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"longevity-tracker/internal/database"
	"longevity-tracker/internal/handlers"
)

func main() {
	// Initialize database
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./health.db"
	}
	db, err := database.Init(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Parse templates with custom functions
	funcMap := template.FuncMap{
		"divf": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"lt": func(a, b float64) bool {
			return a < b
		},
	}
	
	templates := template.Must(template.New("").Funcs(funcMap).ParseGlob("web/templates/*.html"))

	// Initialize handlers
	h := handlers.New(db, templates)

	// Routes
	http.HandleFunc("/", h.HandleHome)
	http.HandleFunc("/current-score", h.HandleCurrentScore)
	http.HandleFunc("/scores", h.HandleScores)
	http.HandleFunc("/calculate-score", h.HandleCalculateScore)
	http.HandleFunc("/update-profile", h.HandleUpdateProfile)
	
	// Metric routes
	http.HandleFunc("/health-metrics", h.HandleHealthMetrics)
	http.HandleFunc("/add-health-metrics", h.HandleAddHealthMetrics)
	http.HandleFunc("/fitness-metrics", h.HandleFitnessMetrics)
	http.HandleFunc("/add-fitness-metrics", h.HandleAddFitnessMetrics)
	http.HandleFunc("/cognition-metrics", h.HandleCognitionMetrics)
	http.HandleFunc("/add-cognition-metrics", h.HandleAddCognitionMetrics)

	// Static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
