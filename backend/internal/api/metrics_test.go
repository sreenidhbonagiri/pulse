package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/sreenidhbonagiri/pulse/backend/internal/cache"
	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
	"github.com/sreenidhbonagiri/pulse/backend/internal/metrics"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
)

func TestMetricsEndpointExposesHTTPCounters(t *testing.T) {
	m := metrics.New("api")
	server := NewServer(
		config.Config{},
		newFakeMonitorRepo(),
		newFakeCheckResultRepo(),
		newFakeIncidentRepo(),
		queue.NewMemoryPublisher(),
		cache.NewMemory(),
	)
	server.metrics = m
	handler := server.Handler()

	health := doRequest(t, handler, http.MethodGet, "/health", "")
	if health.Code != http.StatusOK {
		t.Fatalf("health = %d", health.Code)
	}

	got := doRequest(t, handler, http.MethodGet, "/metrics", "")
	if got.Code != http.StatusOK {
		t.Fatalf("metrics status = %d body=%s", got.Code, got.Body.String())
	}
	body := got.Body.String()
	if !strings.Contains(body, "pulse_http_requests_total") {
		t.Fatalf("missing pulse_http_requests_total:\n%s", body)
	}
	if !strings.Contains(body, `route="/health"`) {
		t.Fatalf("health route missing from metrics:\n%s", body)
	}
}
