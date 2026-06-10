package dashboard

import "net/http"

// securityHeadersMiddleware adds standard security headers to all responses.
// When TLS is active, it also adds Strict-Transport-Security (HSTS).
func securityHeadersMiddleware(tlsEnabled bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-XSS-Protection", "0") // modern browsers rely on CSP, not XSS filter

		if tlsEnabled {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}
