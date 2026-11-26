package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"health-balance/internal/database"
	"health-balance/internal/handlers"
	"health-balance/internal/middleware"
	"health-balance/internal/services"
	"health-balance/internal/ui"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/health.db"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	host := os.Getenv("HOST")

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatalf("failed to create data directory for %s: %v", dbPath, err)
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

	// A nil *Claude must not become a non-nil interface value.
	var insights services.InsightWriter
	if c := services.NewClaudeFromEnv(); c != nil {
		insights = c
	}
	services.StartInsightScheduler(db, insights)

	mux := http.NewServeMux()
	handlers.New(db, insights).Routes(mux)
	mux.Handle("/", ui.Handler(ui.FS()))

	addr := host + ":" + port
	log.Printf("Server starting on %s", addr)
	srv := &http.Server{
		Addr:              addr,
		Handler:           middleware.RequestLogger(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
