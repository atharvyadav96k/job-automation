package api

import (
	"crypto/subtle"
	"net/http"
)

// BasicAuth wraps a handler with HTTP Basic Auth. This is personal profile
// data (and later, API keys) so the API must never be left open, even on a
// server reachable only over a private network.
func BasicAuth(user, pass string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		userMatch := subtle.ConstantTimeCompare([]byte(u), []byte(user)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(p), []byte(pass)) == 1
		if !ok || !userMatch || !passMatch {
			w.Header().Set("WWW-Authenticate", `Basic realm="job-automation"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
