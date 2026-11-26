// Command backfill-health loads history from an Apple Health export
// (export.xml) into health-balance's daily_metrics table.
//
//	go run ./cmd/backfill-health -export ~/Downloads/apple_health_export/export.xml -db ./data/health.db
//
// It overwrites the stored value of every (date, metric) it computes, so it is
// safe to run again. The last day of the export is left out by default
// because it is usually incomplete.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
	_ "time/tzdata" // zone names from the export, also in minimal containers

	"health-balance/internal/database"
	"health-balance/internal/ingest"
	"health-balance/internal/models"
)

func main() {
	var exportPath, dbPath, from, to, tz string
	var dryRun bool
	flag.StringVar(&exportPath, "export", "", "path to the Apple Health export.xml (required)")
	flag.StringVar(&dbPath, "db", "./data/health.db", "SQLite database path")
	flag.StringVar(&from, "from", "2021-10-01", "first date to load (YYYY-MM-DD)")
	flag.StringVar(&to, "to", "", "last date to load (default: the day before the export date)")
	flag.StringVar(&tz, "tz", "Europe/Stockholm", "time zone for records before the first one tagged with a zone")
	flag.BoolVar(&dryRun, "dry-run", false, "parse and report without writing")
	flag.Parse()

	if exportPath == "" {
		flag.Usage()
		os.Exit(2)
	}
	fallback, err := time.LoadLocation(tz)
	if err != nil {
		log.Fatal(err)
	}
	if err := run(exportPath, dbPath, from, to, fallback, dryRun); err != nil {
		log.Fatal(err)
	}
}

func run(exportPath, dbPath, from, to string, fallback *time.Location, dryRun bool) error {
	started := time.Now()

	f, err := os.Open(exportPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	var samples []ingest.Sample
	stats, err := ingest.ReadExport(bufio.NewReaderSize(f, 1<<20), fallback, func(s ingest.Sample) {
		// Keep the evening before `from` so the first night is complete.
		if d := s.Start.Format("2006-01-02"); d >= dayBefore(from) && (to == "" || d <= to) {
			samples = append(samples, s)
		}
	})
	if err != nil {
		return err
	}
	if to == "" {
		if stats.ExportDate.IsZero() {
			return fmt.Errorf("export has no ExportDate; pass -to")
		}
		to = stats.ExportDate.AddDate(0, 0, -1).Format("2006-01-02")
	}
	log.Printf("Read %d records in %v; export date %s; using %d samples",
		stats.Records, time.Since(started).Round(time.Millisecond), stats.ExportDate.Format("2006-01-02"), len(samples))
	for reason, n := range stats.Rejected {
		log.Printf("  rejected %6d × %s", n, reason)
	}
	for zone, n := range stats.Zones {
		log.Printf("  %7d samples in %s", n, zone)
	}

	var values []models.DailyMetric
	perMetric := map[string][2]string{}
	counts := map[string]int{}
	before := len(samples)
	samples = dedupeSources(samples)
	log.Printf("Removed %d duplicate-source samples", before-len(samples))
	for _, dv := range ingest.Aggregate(samples, ingest.Options{}) {
		if dv.Date < from || dv.Date > to {
			continue
		}
		values = append(values, models.DailyMetric{Date: dv.Date, Metric: dv.Metric, Value: dv.Value, Samples: dv.Samples, Source: dv.Source})
		counts[dv.Metric]++
		r := perMetric[dv.Metric]
		if r[0] == "" || dv.Date < r[0] {
			r[0] = dv.Date
		}
		if dv.Date > r[1] {
			r[1] = dv.Date
		}
		perMetric[dv.Metric] = r
	}

	metrics := make([]string, 0, len(counts))
	for m := range counts {
		metrics = append(metrics, m)
	}
	sort.Strings(metrics)
	fmt.Printf("%-18s %6s  %s\n", "metric", "days", "range")
	for _, m := range metrics {
		fmt.Printf("%-18s %6d  %s … %s\n", m, counts[m], perMetric[m][0], perMetric[m][1])
	}

	if dryRun {
		log.Printf("Dry run: %d daily values not written", len(values))
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return err
	}
	db, err := database.Init(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	rejected := 0
	for _, n := range stats.Rejected {
		rejected += n
	}
	id, err := db.SaveIngest(models.IngestRun{
		Kind:     models.IngestKindBackfill,
		Rows:     len(samples),
		Accepted: len(values),
		Rejected: rejected,
		FromDate: from,
		ToDate:   to,
	}, values, nil)
	if err != nil {
		return err
	}
	log.Printf("Wrote %d daily values (%s … %s) to %s as run %d in %v",
		len(values), from, to, dbPath, id, time.Since(started).Round(time.Millisecond))
	return nil
}

func dayBefore(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, -1).Format("2006-01-02")
}
