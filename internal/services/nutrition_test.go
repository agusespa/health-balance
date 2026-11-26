package services

import (
	"testing"
	"time"

	"health-balance/internal/ingest"
	"health-balance/internal/models"
)

func TestGetNutritionSummary(t *testing.T) {
	v := func(date, metric string, value float64) models.DailyMetric {
		return models.DailyMetric{Date: date, Metric: metric, Value: value}
	}
	db := scoreDB(t, []models.DailyMetric{
		v("2026-10-01", ingest.MetricWeightKg, 80),
		// Two full days with burn data.
		v("2026-11-20", ingest.MetricEnergyIn, 2000),
		v("2026-11-20", ingest.MetricProtein, 150),
		v("2026-11-20", ingest.MetricCarbs, 200),
		v("2026-11-20", ingest.MetricFat, 70),
		v("2026-11-20", ingest.MetricFiber, 30),
		v("2026-11-20", ingest.MetricActiveEnergy, 600),
		v("2026-11-20", ingest.MetricRestingEnergy, 1800),
		v("2026-11-21", ingest.MetricEnergyIn, 2600),
		v("2026-11-21", ingest.MetricProtein, 130),
		v("2026-11-21", ingest.MetricCarbs, 300),
		v("2026-11-21", ingest.MetricFat, 90),
		v("2026-11-21", ingest.MetricFiber, 24),
		v("2026-11-21", ingest.MetricActiveEnergy, 400),
		v("2026-11-21", ingest.MetricRestingEnergy, 1800),
		// A full day without burn data counts for intake only.
		v("2026-11-22", ingest.MetricEnergyIn, 2300),
		v("2026-11-22", ingest.MetricProtein, 140),
		// An unfinished diary is left out; its water still counts.
		v("2026-11-23", ingest.MetricEnergyIn, 450),
		v("2026-11-23", ingest.MetricProtein, 30),
		v("2026-11-23", ingest.MetricWater, 2000),
		v("2026-11-24", ingest.MetricWater, 1000),
		// Outside the window.
		v("2026-11-27", ingest.MetricEnergyIn, 5000),
	})

	got, err := GetNutritionSummary(db, time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	want := NutritionSummary{
		From: "2026-11-20", To: "2026-11-26", Days: 7, LoggedDays: 3,
		EnergyInKcal: 2300, ProteinG: 140, CarbsG: 167, FatG: 53, FiberG: 18, ProteinPerKg: 1.8,
		BalanceDays: 2, BurnedKcal: 2300, BalanceKcal: 0,
		WaterDays: 2, WaterMl: 1500,
	}
	if got == nil || *got != want {
		t.Errorf("got %+v\nwant %+v", got, want)
	}

	none, err := GetNutritionSummary(db, time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC))
	if err != nil || none != nil {
		t.Errorf("empty week = %+v, %v; want nil", none, err)
	}
}
