package services

import (
	"fmt"
	"time"

	"health-balance/internal/database"
	"health-balance/internal/ingest"
)

const (
	// NutritionDays is how many days the nutrition summary covers.
	NutritionDays = 7
	// MinLoggedKcal is the energy a day needs before it counts as logged; a
	// day below it is most likely a diary left unfinished.
	MinLoggedKcal = 1000.0
	// weightLookbackDays is how far back the latest weight is looked for,
	// to express protein per kilogram.
	weightLookbackDays = 90
)

// NutritionSummary is the daily average over the days of a period that have
// food logged. It is shown only and does not affect the score.
type NutritionSummary struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Days       int    `json:"days"`
	LoggedDays int    `json:"logged_days"` // days with at least MinLoggedKcal

	EnergyInKcal float64 `json:"energy_in_kcal"`
	ProteinG     float64 `json:"protein_g"`
	CarbsG       float64 `json:"carbs_g"`
	FatG         float64 `json:"fat_g"`
	FiberG       float64 `json:"fiber_g"`
	ProteinPerKg float64 `json:"protein_per_kg"` // 0 without a recent weight

	// Energy balance: eaten minus active and resting energy burned, over the
	// logged days that also have both burn values. Negative is a deficit.
	BalanceDays int     `json:"balance_days"`
	BurnedKcal  float64 `json:"burned_kcal"`
	BalanceKcal float64 `json:"balance_kcal"`

	// Water is averaged over the days it was logged, logged food or not.
	WaterDays int     `json:"water_days"`
	WaterMl   float64 `json:"water_ml"`
}

// GetNutritionSummary summarises the NutritionDays days ending on to. It
// returns nil when no nutrition at all was logged in that time.
func GetNutritionSummary(db database.Querier, to time.Time) (*NutritionSummary, error) {
	from := to.AddDate(0, 0, -(NutritionDays - 1))
	toS := to.Format(dateLayout)
	rows, err := db.GetDailyMetrics(from.AddDate(0, 0, -weightLookbackDays).Format(dateLayout), toS,
		ingest.MetricEnergyIn, ingest.MetricProtein, ingest.MetricCarbs, ingest.MetricFat, ingest.MetricFiber, ingest.MetricWater,
		ingest.MetricActiveEnergy, ingest.MetricRestingEnergy, ingest.MetricWeightKg)
	if err != nil {
		return nil, fmt.Errorf("load nutrition: %w", err)
	}
	s := newSeries(rows)

	sum := NutritionSummary{From: from.Format(dateLayout), To: toS, Days: NutritionDays}
	anything := false
	var burned, balance float64
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		date := d.Format(dateLayout)
		day := func(metric string) (float64, bool) {
			v, ok := s.byMetric[metric][date]
			return v, ok
		}
		if v, ok := day(ingest.MetricWater); ok {
			sum.WaterDays++
			sum.WaterMl += v
			anything = true
		}
		kcal, ok := day(ingest.MetricEnergyIn)
		if ok {
			anything = true
		}
		if kcal < MinLoggedKcal {
			continue
		}
		sum.LoggedDays++
		sum.EnergyInKcal += kcal
		p, _ := day(ingest.MetricProtein)
		c, _ := day(ingest.MetricCarbs)
		f, _ := day(ingest.MetricFat)
		fib, _ := day(ingest.MetricFiber)
		sum.ProteinG += p
		sum.CarbsG += c
		sum.FatG += f
		sum.FiberG += fib
		active, okA := day(ingest.MetricActiveEnergy)
		resting, okR := day(ingest.MetricRestingEnergy)
		if okA && okR {
			sum.BalanceDays++
			burned += active + resting
			balance += kcal - active - resting
		}
	}
	if !anything {
		return nil, nil
	}

	if n := float64(sum.LoggedDays); n > 0 {
		sum.EnergyInKcal = round(sum.EnergyInKcal/n, 0)
		sum.ProteinG = round(sum.ProteinG/n, 0)
		sum.CarbsG = round(sum.CarbsG/n, 0)
		sum.FatG = round(sum.FatG/n, 0)
		sum.FiberG = round(sum.FiberG/n, 0)
		if kg, ok := s.last(ingest.MetricWeightKg, from.AddDate(0, 0, -weightLookbackDays).Format(dateLayout), toS); ok && kg > 0 {
			sum.ProteinPerKg = round(sum.ProteinG/kg, 1)
		}
	}
	if n := float64(sum.BalanceDays); n > 0 {
		sum.BurnedKcal = round(burned/n, 0)
		sum.BalanceKcal = round(balance/n, 0)
	}
	if n := float64(sum.WaterDays); n > 0 {
		sum.WaterMl = round(sum.WaterMl/n, 0)
	}
	return &sum, nil
}
