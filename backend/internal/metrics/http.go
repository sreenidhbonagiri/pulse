package metrics

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var uuidPath = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

// MetricsHandler serves GET /metrics from the process registry.
func MetricsHandler(m *Metrics) http.Handler {
	if m == nil || m.Gatherer() == nil {
		return http.NotFoundHandler()
	}
	return promhttp.HandlerFor(m.Gatherer(), promhttp.HandlerOpts{})
}

// HTTPMiddleware records request count, in-flight gauge, and duration.
// Route labels use the ServeMux pattern (for example /api/monitors/{id}),
// never the raw UUID path.
func HTTPMiddleware(m *Metrics, next http.Handler) http.Handler {
	if m == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		m.HTTPInFlightInc()
		defer m.HTTPInFlightDec()

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)
		route := NormalizeRoute(r.Pattern, r.URL.Path)
		m.ObserveHTTP(r.Method, route, StatusClass(rec.status), time.Since(start))
	})
}

func StatusClass(status int) string {
	switch {
	case status >= 100 && status < 200:
		return "1xx"
	case status >= 200 && status < 300:
		return "2xx"
	case status >= 300 && status < 400:
		return "3xx"
	case status >= 400 && status < 500:
		return "4xx"
	default:
		return "5xx"
	}
}

// NormalizeRoute returns a bounded HTTP route label.
func NormalizeRoute(pattern, path string) string {
	if route := routeFromPattern(pattern); route != "" {
		return route
	}
	if path == "" {
		return "unmatched"
	}
	cleaned := uuidPath.ReplaceAllString(path, "{id}")
	if cleaned != path {
		return cleaned
	}
	return "unmatched"
}

func routeFromPattern(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return ""
	}
	method, route, ok := strings.Cut(pattern, " ")
	if !ok {
		return pattern
	}
	if method == "" || route == "" {
		return pattern
	}
	return route
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecorder) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}
