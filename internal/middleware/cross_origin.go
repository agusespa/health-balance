package middleware

import "net/http"

// CrossOrigin refuses requests that change data when a browser marks them as
// coming from another site. The app has no login of its own and relies on
// Cloudflare Access, so this stops a page you visit from using your Access
// session to change your data. Scripts, n8n and MCP clients send no such
// browser headers and are not affected.
func CrossOrigin(next http.Handler) http.Handler {
	csrf := http.NewCrossOriginProtection()
	csrf.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"cross-origin request refused"}` + "\n"))
	}))
	return csrf.Handler(next)
}
