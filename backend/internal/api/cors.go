package api

import (
	"net/http"
	"strings"
)

func corsMiddleware(allowedOrigins string, next http.Handler) http.Handler {
	allowed := parseOrigins(allowedOrigins)
	if len(allowed) == 0 {
		allowed = []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowOrigin, ok := matchOrigin(allowed, origin); ok {
			w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			w.Header().Add("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func parseOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}

func matchOrigin(allowed []string, origin string) (string, bool) {
	if origin == "" {
		return "", false
	}
	for _, candidate := range allowed {
		if candidate == "*" || candidate == origin {
			return origin, true
		}
	}
	return "", false
}
