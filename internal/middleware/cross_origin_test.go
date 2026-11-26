package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCrossOrigin(t *testing.T) {
	h := CrossOrigin(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	tests := []struct {
		name, method string
		headers      map[string]string
		want         int
	}{
		{"script without browser headers", http.MethodPost, nil, http.StatusNoContent},
		{"same-origin browser write", http.MethodPut, map[string]string{"Sec-Fetch-Site": "same-origin"}, http.StatusNoContent},
		{"cross-site browser write", http.MethodPost, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"cross-origin Origin header", http.MethodPost, map[string]string{"Origin": "https://evil.example"}, http.StatusForbidden},
		{"cross-site read", http.MethodGet, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusNoContent},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "http://health.example/api/ingest/health", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Errorf("status %d, want %d", rec.Code, tc.want)
			}
		})
	}
}
