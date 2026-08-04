package httpapi

import "net/http"

// WithCORS allows the local React dev server (a different origin/port) to
// call this API directly from the browser. Wide open ("*") is a deliberate
// local-dev-only simplification — a real deployment would restrict this to
// the actual UI's origin, not allow any site to call the API.
func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
