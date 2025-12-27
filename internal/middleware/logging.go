package middleware

import (
	"log"
	"net/http"
	"time"
)

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		sw := NewStatusResponseWriter(w)

		next.ServeHTTP(sw, r)

		log.Printf("%s %s %d %v", r.Method, r.URL.Path, sw.statusCode, time.Since(start))
	})
}
