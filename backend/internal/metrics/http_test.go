package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestNormalizeRouteUsesPatternNotUUID(t *testing.T) {
	path := "/api/monitors/3e30af9e-6c07-4fd9-abd1-b7d370b82849/stats"
	got := NormalizeRoute("GET /api/monitors/{id}/stats", path)
	if got != "/api/monitors/{id}/stats" {
		t.Fatalf("route = %q", got)
	}
}

func TestNormalizeRouteStripsUUIDWhenPatternMissing(t *testing.T) {
	got := NormalizeRoute("", "/api/monitors/3e30af9e-6c07-4fd9-abd1-b7d370b82849")
	if got != "/api/monitors/{id}" {
		t.Fatalf("route = %q, want /api/monitors/{id}", got)
	}
}

func TestNormalizeRouteUnknownPathIsBounded(t *testing.T) {
	got := NormalizeRoute("", "/totally/unknown/"+strings.Repeat("x", 40))
	if got != "unmatched" {
		t.Fatalf("route = %q, want unmatched so labels stay bounded", got)
	}
}

func TestHTTPMiddlewareRecordsNormalizedRoute(t *testing.T) {
	m := New("api")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/monitors/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/monitors/3e30af9e-6c07-4fd9-abd1-b7d370b82849", nil)
	rec := httptest.NewRecorder()
	HTTPMiddleware(m, mux).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	body := gatherMetric(t, m, "pulse_http_requests_total")
	if strings.Contains(body, "3e30af9e-6c07-4fd9-abd1-b7d370b82849") {
		t.Fatalf("metrics leaked a UUID:\n%s", body)
	}
	if !strings.Contains(body, `value:"/api/monitors/{id}"`) {
		t.Fatalf("missing normalized route:\n%s", body)
	}
	if !strings.Contains(body, `value:"GET"`) || !strings.Contains(body, `value:"2xx"`) {
		t.Fatalf("missing method/status labels:\n%s", body)
	}

	inFlight := gaugeValue(t, m, "pulse_http_requests_in_flight")
	if inFlight != 0 {
		t.Fatalf("in-flight = %v, want 0 after the request", inFlight)
	}
}

func TestAdminMuxHealth(t *testing.T) {
	m := New("worker")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	AdminMux(m).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHTTPMiddlewareSkipsMetricsPath(t *testing.T) {
	m := New("api")
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", MetricsHandler(m))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	HTTPMiddleware(m, mux).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	body := gatherMetric(t, m, "pulse_http_requests_total")
	if strings.Contains(body, `/metrics`) {
		t.Fatalf("scrapes should not record HTTP request metrics:\n%s", body)
	}
}

func gatherMetric(t *testing.T, m *Metrics, name string) string {
	t.Helper()
	families, err := m.Gatherer().Gather()
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.Metric {
			b.WriteString(metric.String())
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func gaugeValue(t *testing.T, m *Metrics, name string) float64 {
	t.Helper()
	families, err := m.Gatherer().Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		if len(family.Metric) == 0 || family.Metric[0].Gauge == nil {
			return 0
		}
		return family.Metric[0].Gauge.GetValue()
	}
	return 0
}

func counterValue(t *testing.T, m *Metrics, name string, labels prometheus.Labels) float64 {
	t.Helper()
	families, err := m.Gatherer().Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.Metric {
			if labelsMatch(metric, labels) && metric.Counter != nil {
				return metric.Counter.GetValue()
			}
		}
	}
	return 0
}

func labelsMatch(metric *dto.Metric, want prometheus.Labels) bool {
	got := map[string]string{}
	for _, label := range metric.Label {
		got[label.GetName()] = label.GetValue()
	}
	for key, value := range want {
		if got[key] != value {
			return false
		}
	}
	return true
}

func TestStatusClass(t *testing.T) {
	if StatusClass(201) != "2xx" || StatusClass(404) != "4xx" || StatusClass(500) != "5xx" {
		t.Fatal("unexpected status class")
	}
}
