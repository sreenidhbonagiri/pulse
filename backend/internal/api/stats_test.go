package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

func TestGetMonitorStats(t *testing.T) {
	env := newTestEnv()
	monitor := createJSONMonitor(t, env.handler, "https://example.com/health", http.StatusOK)
	now := time.Now().UTC()

	ok := 200
	fail := 500
	results := []models.CheckResult{
		{MonitorID: monitor.ID, StatusCode: &ok, ResponseTimeMs: 10, Success: true, CheckedAt: now.Add(-4 * time.Minute)},
		{MonitorID: monitor.ID, StatusCode: &ok, ResponseTimeMs: 20, Success: true, CheckedAt: now.Add(-3 * time.Minute)},
		{MonitorID: monitor.ID, StatusCode: &fail, ResponseTimeMs: 30, Success: false, CheckedAt: now.Add(-2 * time.Minute)},
		{MonitorID: monitor.ID, StatusCode: &ok, ResponseTimeMs: 40, Success: true, CheckedAt: now.Add(-time.Minute)},
		{MonitorID: monitor.ID, StatusCode: &ok, ResponseTimeMs: 50, Success: true, CheckedAt: now},
	}
	for i := range results {
		if err := env.checkResults.Create(nil, &results[i]); err != nil {
			t.Fatal(err)
		}
	}
	if err := env.incidents.Create(nil, &models.Incident{
		MonitorID:    monitor.ID,
		StartedAt:    now.Add(-time.Hour),
		Status:       models.IncidentStatusOpen,
		FailureCount: 3,
	}); err != nil {
		t.Fatal(err)
	}

	rec := doRequest(t, env.handler, http.MethodGet, "/api/monitors/"+monitor.ID.String()+"/stats", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var stats models.MonitorStats
	decodeBody(t, rec, &stats)
	if stats.CurrentStatus != models.MonitorStatusUp {
		t.Fatalf("current_status = %s", stats.CurrentStatus)
	}
	if stats.TotalChecks != 5 || stats.FailedChecks != 1 || stats.UptimePercentage != 80 {
		t.Fatalf("stats = %+v", stats)
	}
	if !stats.ActiveIncident || stats.IncidentCount != 1 {
		t.Fatalf("incidents active=%t count=%d", stats.ActiveIncident, stats.IncidentCount)
	}
}

func TestGetMonitorStatsUnknownMonitor(t *testing.T) {
	handler := newTestHandler()
	rec := doRequest(t, handler, http.MethodGet, "/api/monitors/"+uuid.NewString()+"/stats", "")
	assertErrorStatus(t, rec, http.StatusNotFound, "monitor not found")
}

func TestGetMonitorStatsEmpty(t *testing.T) {
	env := newTestEnv()
	monitor := createJSONMonitor(t, env.handler, "https://example.com/health", http.StatusOK)

	rec := doRequest(t, env.handler, http.MethodGet, "/api/monitors/"+monitor.ID.String()+"/stats", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var stats models.MonitorStats
	decodeBody(t, rec, &stats)
	if stats.CurrentStatus != models.MonitorStatusUnknown || stats.TotalChecks != 0 {
		t.Fatalf("empty stats = %+v", stats)
	}
}
