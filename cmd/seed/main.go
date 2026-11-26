// Command seed fills a database with a profile and plausible synthetic daily
// Health data, for local development and the e2e tests.
//
//	go run ./cmd/seed -db ./data/dev.db -days 900 -reset
package main

import (
	"flag"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"time"

	"health-balance/internal/database"
	"health-balance/internal/ingest"
	"health-balance/internal/models"
)

func main() {
	var dbPath string
	var days int
	var reset bool
	flag.StringVar(&dbPath, "db", "./data/health.db", "SQLite database path")
	flag.IntVar(&days, "days", 900, "days of history to generate, ending yesterday")
	flag.BoolVar(&reset, "reset", false, "delete the profile and all automatic data first")
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatal(err)
	}
	db, err := database.Init(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	if reset {
		for _, table := range []string{"user_profile", "daily_metrics", "ingest_runs", "insights"} {
			if _, err := db.Exec("DELETE FROM " + table); err != nil {
				log.Fatalf("reset %s: %v", table, err)
			}
		}
	}

	if err := db.SaveUserProfile(models.UserProfile{BirthDate: "1990-06-15", Sex: "male", HeightCm: 180}); err != nil {
		log.Fatal(err)
	}

	today := time.Now()
	values := generate(today, days, rand.New(rand.NewPCG(42, 7)))
	run := models.IngestRun{
		ReceivedAt: today.Add(-6 * time.Hour),
		Kind:       models.IngestKindShortcut,
		Rows:       len(values),
		Accepted:   len(values),
		FromDate:   values[0].Date,
		ToDate:     values[len(values)-1].Date,
	}
	if _, err := db.SaveIngest(run, values, nil); err != nil {
		log.Fatal(err)
	}
	log.Printf("Seeded %d daily values over %d days into %s", len(values), days, dbPath)
}

// generate makes daily values that slowly improve toward the present, with
// day-to-day noise, sparse markers (VO2 every ~2 weeks, waist and weight
// monthly, blood pressure three mornings a quarter), gym sets for the last 3
// weeks and food for the last 30 days.
func generate(today time.Time, days int, r *rand.Rand) []models.DailyMetric {
	var out []models.DailyMetric
	add := func(date, metric string, v float64) {
		out = append(out, models.DailyMetric{Date: date, Metric: metric, Value: math.Round(v*10) / 10, Samples: 1, Source: "watch"})
	}
	for i := days; i >= 1; i-- {
		day := today.AddDate(0, 0, -i)
		date := day.Format("2006-01-02")
		progress := 1 - float64(i)/float64(days) // 0 → 1 over the period
		noise := func(sd float64) float64 { return r.NormFloat64() * sd }

		add(date, ingest.MetricRHR, 56-4*progress+noise(2))
		add(date, ingest.MetricHRV, 52+10*progress+noise(9))
		add(date, ingest.MetricSteps, math.Max(1500, 6500+2500*progress+noise(2500)))
		add(date, ingest.MetricExerciseMin, math.Max(0, 30+20*progress+noise(18)))
		add(date, ingest.MetricSleepHours, 7.2+0.4*progress+noise(0.6))
		add(date, ingest.MetricSleepBedtime, 285+noise(25)) // about 22:45
		add(date, ingest.MetricSleepAwakeMin, math.Max(0, 8+noise(5)))
		add(date, ingest.MetricSleepAwakenings, math.Max(0, math.Round(2+noise(1.2))))

		if i%14 == 0 {
			add(date, ingest.MetricVO2Max, 40+4*progress+noise(0.6))
		}
		if i%30 == 0 {
			add(date, ingest.MetricWaistCm, 88-5*progress+noise(0.5))
			add(date, ingest.MetricWeightKg, 82-3*progress+noise(0.4))
		}
		if i%91 < 3 {
			add(date, ingest.MetricBPSys, 120-6*progress+noise(3))
			add(date, ingest.MetricBPDia, 79-5*progress+noise(2))
		}
		if i%21 == 0 {
			add(date, ingest.MetricCardioRecovery, 24+6*progress+noise(2))
		}
		if day.Weekday() == time.Sunday || day.Weekday() == time.Wednesday {
			add(date, ingest.MetricMindfulSessions, 1)
			add(date, ingest.MetricMindfulMin, 10)
		}
		add(date, ingest.MetricActiveEnergy, math.Max(150, 450+150*progress+noise(120)))
		add(date, ingest.MetricRestingEnergy, 1750+noise(30))
		// Food logged for the last 30 days, with a skipped day each week and
		// an unfinished one every 9 days.
		if i <= 30 && i%7 != 3 {
			kcal := 2350 + noise(250)
			if i%9 == 0 {
				kcal = 650
			}
			out = append(out,
				models.DailyMetric{Date: date, Metric: ingest.MetricEnergyIn, Value: math.Round(kcal), Samples: 4, Source: "Cronometer"},
				models.DailyMetric{Date: date, Metric: ingest.MetricProtein, Value: math.Round(kcal * 0.065), Samples: 4, Source: "Cronometer"},
				models.DailyMetric{Date: date, Metric: ingest.MetricCarbs, Value: math.Round(kcal * 0.11), Samples: 4, Source: "Cronometer"},
				models.DailyMetric{Date: date, Metric: ingest.MetricFat, Value: math.Round(kcal * 0.035), Samples: 4, Source: "Cronometer"},
				models.DailyMetric{Date: date, Metric: ingest.MetricFiber, Value: math.Round(kcal * 0.012), Samples: 4, Source: "Cronometer"},
				models.DailyMetric{Date: date, Metric: ingest.MetricWater, Value: math.Round(1800 + noise(400)), Samples: 6, Source: "Cronometer"},
			)
		}
		if i <= 21 {
			sets := 0.0
			switch day.Weekday() {
			case time.Monday, time.Thursday:
				sets = 7
			}
			out = append(out, models.DailyMetric{Date: date, Metric: ingest.MetricWorkingSets, Value: sets, Samples: 1, Source: "gympass"})
		}
	}
	return out
}
