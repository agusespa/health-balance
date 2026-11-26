package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"health-balance/internal/models"
)

const timeLayout = time.RFC3339Nano

// SaveIngest upserts the daily values, deletes the clear ones and records the
// run in one transaction. A value for an existing (date, metric) replaces it,
// so re-sending the same days is safe; days and metrics in neither values nor
// clear are left untouched. It returns the run's id.
func (db *DB) SaveIngest(run models.IngestRun, values []models.DailyMetric, clear []models.DayMetric) (int64, error) {
	if run.ReceivedAt.IsZero() {
		run.ReceivedAt = time.Now()
	}
	receivedAt := run.ReceivedAt.UTC().Format(timeLayout)

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			log.Printf("error rolling back ingest: %v", err)
		}
	}()

	stmt, err := tx.Prepare(`
		INSERT INTO daily_metrics (date, metric, value, samples, source, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(date, metric) DO UPDATE SET
			value = excluded.value,
			samples = excluded.samples,
			source = excluded.source,
			updated_at = excluded.updated_at`)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Printf("error closing upsert statement: %v", err)
		}
	}()

	for _, v := range values {
		if _, err := stmt.Exec(v.Date, v.Metric, v.Value, v.Samples, v.Source, receivedAt); err != nil {
			return 0, fmt.Errorf("upsert %s %s: %w", v.Date, v.Metric, err)
		}
	}

	for _, c := range clear {
		if _, err := tx.Exec(`DELETE FROM daily_metrics WHERE date = ? AND metric = ?`, c.Date, c.Metric); err != nil {
			return 0, fmt.Errorf("clear %s %s: %w", c.Date, c.Metric, err)
		}
	}

	res, err := tx.Exec(`
		INSERT INTO ingest_runs (received_at, kind, rows, accepted, rejected, from_date, to_date)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		receivedAt, run.Kind, run.Rows, run.Accepted, run.Rejected, run.FromDate, run.ToDate)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	_, _ = db.Exec("PRAGMA wal_checkpoint(PASSIVE)")
	return id, nil
}

// GetDailyMetrics returns values with from <= date <= to, ordered by date and
// metric. With no metrics given it returns all of them.
func (db *DB) GetDailyMetrics(from, to string, metrics ...string) ([]models.DailyMetric, error) {
	query := `SELECT date, metric, value, samples, source, updated_at FROM daily_metrics WHERE date >= ? AND date <= ?`
	args := []any{from, to}
	if len(metrics) > 0 {
		query += ` AND metric IN (?` + strings.Repeat(`, ?`, len(metrics)-1) + `)`
		for _, m := range metrics {
			args = append(args, m)
		}
	}
	query += ` ORDER BY date, metric`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("error closing rows: %v", err)
		}
	}()

	var out []models.DailyMetric
	for rows.Next() {
		var m models.DailyMetric
		var updatedAt string
		if err := rows.Scan(&m.Date, &m.Metric, &m.Value, &m.Samples, &m.Source, &updatedAt); err != nil {
			return nil, err
		}
		if m.UpdatedAt, err = time.Parse(timeLayout, updatedAt); err != nil {
			return nil, fmt.Errorf("daily_metrics %s %s: %w", m.Date, m.Metric, err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetLastIngestRun returns the most recent run of the given kind, or nil.
func (db *DB) GetLastIngestRun(kind string) (*models.IngestRun, error) {
	var r models.IngestRun
	var receivedAt string
	err := db.QueryRow(`
		SELECT id, received_at, kind, rows, accepted, rejected, from_date, to_date
		FROM ingest_runs WHERE kind = ?
		ORDER BY received_at DESC, id DESC LIMIT 1`, kind).
		Scan(&r.Id, &receivedAt, &r.Kind, &r.Rows, &r.Accepted, &r.Rejected, &r.FromDate, &r.ToDate)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if r.ReceivedAt, err = time.Parse(timeLayout, receivedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

// SaveInsight stores (or replaces) the insight for a week.
func (db *DB) SaveInsight(in models.Insight) error {
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}
	_, err := db.Exec(`
		INSERT INTO insights (week_end, model, created_at, markdown) VALUES (?, ?, ?, ?)
		ON CONFLICT(week_end) DO UPDATE SET
			model = excluded.model,
			created_at = excluded.created_at,
			markdown = excluded.markdown`,
		in.WeekEnd, in.Model, in.CreatedAt.UTC().Format(timeLayout), in.Markdown)
	return err
}

// GetLatestInsight returns the insight for the most recent week, or nil.
func (db *DB) GetLatestInsight() (*models.Insight, error) {
	var in models.Insight
	var createdAt string
	err := db.QueryRow(`SELECT week_end, model, created_at, markdown FROM insights ORDER BY week_end DESC LIMIT 1`).
		Scan(&in.WeekEnd, &in.Model, &createdAt, &in.Markdown)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if in.CreatedAt, err = time.Parse(timeLayout, createdAt); err != nil {
		return nil, err
	}
	return &in, nil
}
