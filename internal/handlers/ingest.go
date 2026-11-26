package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"health-balance/internal/services"
)

// maxIngestBody caps a Shortcut payload; seven days of samples are well under 1 MB.
const maxIngestBody = 5 << 20

type ingestRequest struct {
	Rows string `json:"rows"`
}

type ingestError struct {
	Error string `json:"error"`
	*services.IngestResult
}

// HandleIngestHealth accepts the n8n relay of the iPhone Shortcut payload,
// {"rows": "KEY;value;unit;start;end;source\n..."}, authenticated with
// "Authorization: Bearer $INGEST_TOKEN". It responds with what was stored.
func (h *Handler) HandleIngestHealth(w http.ResponseWriter, r *http.Request) {
	token := os.Getenv("INGEST_TOKEN")
	if token == "" {
		writeJSON(w, http.StatusServiceUnavailable, ingestError{Error: "ingest is disabled: INGEST_TOKEN is not set"})
		return
	}
	if !validBearer(r.Header.Get("Authorization"), token) {
		writeJSON(w, http.StatusUnauthorized, ingestError{Error: "unauthorized"})
		return
	}

	var req ingestRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxIngestBody))
	if err := dec.Decode(&req); err != nil {
		status := http.StatusBadRequest
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			status = http.StatusRequestEntityTooLarge
		}
		writeJSON(w, status, ingestError{Error: "invalid JSON body"})
		return
	}

	res, err := services.IngestShortcutRows(h.db, req.Rows, time.Now())
	switch {
	case errors.Is(err, services.ErrNoUsableRows):
		writeJSON(w, http.StatusBadRequest, ingestError{Error: err.Error(), IngestResult: &res})
	case err != nil:
		log.Printf("Ingest error: %v", err)
		writeJSON(w, http.StatusInternalServerError, ingestError{Error: "could not store the data"})
	default:
		log.Printf("Ingest: %d rows, %d daily values, %d rejected, %s..%s", res.Rows, res.Accepted, res.Rejected, res.From, res.To)
		writeJSON(w, http.StatusOK, res)
	}
}

func validBearer(header, token string) bool {
	got, ok := strings.CutPrefix(header, "Bearer ")
	return ok && subtle.ConstantTimeCompare([]byte(strings.TrimSpace(got)), []byte(token)) == 1
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}
