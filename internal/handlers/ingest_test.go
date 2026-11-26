package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"health-balance/internal/models"
)

const rhrBody = `{"rows":"RHR; 53;count/min;2026-11-18T00:03:14+01:00;2026-11-18T23:54:13+01:00;Agustin’s Apple Watch\nRHR; 55;count/min;2026-11-19T00:01:30+01:00;2026-11-19T23:54:46+01:00;Agustin’s Apple Watch"}`

func ingestRequestWith(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/ingest/health", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestHandleIngestHealthStores(t *testing.T) {
	h, mock := setupTestHandler()
	var saved []models.DailyMetric
	var savedRun models.IngestRun
	mock.SaveIngestFunc = func(run models.IngestRun, values []models.DailyMetric, _ []models.DayMetric) (int64, error) {
		savedRun, saved = run, values
		return 7, nil
	}

	rec := httptest.NewRecorder()
	h.HandleIngestHealth(rec, ingestRequestWith(rhrBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var res struct {
		RunId    int64          `json:"run_id"`
		Accepted int            `json:"accepted"`
		Metrics  map[string]int `json:"metrics"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.RunId != 7 || res.Accepted != 2 || res.Metrics["rhr"] != 2 {
		t.Errorf("response %s", rec.Body)
	}
	if len(saved) != 2 || saved[0].Date != "2026-11-18" || saved[0].Value != 53 || savedRun.Kind != models.IngestKindShortcut {
		t.Errorf("saved %+v run %+v", saved, savedRun)
	}
}

func TestHandleIngestHealthErrors(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		saveErr error
		want    int
	}{
		{"invalid JSON", `{"rows":`, nil, http.StatusBadRequest},
		{"no usable rows", `{"rows":"garbage"}`, nil, http.StatusBadRequest},
		{"too large", `{"rows":"` + strings.Repeat("x", maxIngestBody) + `"}`, nil, http.StatusRequestEntityTooLarge},
		{"db failure", rhrBody, errors.New("disk full"), http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, mock := setupTestHandler()
			mock.SaveIngestFunc = func(models.IngestRun, []models.DailyMetric, []models.DayMetric) (int64, error) { return 1, tc.saveErr }

			rec := httptest.NewRecorder()
			h.HandleIngestHealth(rec, ingestRequestWith(tc.body))
			if rec.Code != tc.want {
				t.Errorf("status %d, want %d: %s", rec.Code, tc.want, rec.Body)
			}
			if strings.Contains(rec.Body.String(), "disk full") {
				t.Error("internal error leaked to the client")
			}
		})
	}
}
