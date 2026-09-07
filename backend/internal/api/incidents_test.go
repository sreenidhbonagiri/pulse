package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

func TestListMonitorIncidentsNewestFirst(t *testing.T) {
	env := newTestEnv()
	monitor := createJSONMonitor(t, env.handler, "https://example.com/health", http.StatusOK)

	older := models.Incident{
		MonitorID:    monitor.ID,
		StartedAt:    time.Now().UTC().Add(-time.Hour),
		Status:       models.IncidentStatusResolved,
		FailureCount: 3,
	}
	resolved := time.Now().UTC().Add(-30 * time.Minute)
	older.ResolvedAt = &resolved
	newer := models.Incident{
		MonitorID:    monitor.ID,
		StartedAt:    time.Now().UTC(),
		Status:       models.IncidentStatusOpen,
		FailureCount: 4,
	}
	if err := env.incidents.Create(nil, &older); err != nil {
		t.Fatal(err)
	}
	if err := env.incidents.Create(nil, &newer); err != nil {
		t.Fatal(err)
	}

	rec := doRequest(t, env.handler, http.MethodGet, "/api/monitors/"+monitor.ID.String()+"/incidents", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var listed []models.Incident
	decodeBody(t, rec, &listed)
	if len(listed) != 2 {
		t.Fatalf("len = %d, want 2", len(listed))
	}
	if listed[0].ID != newer.ID || listed[1].ID != older.ID {
		t.Fatalf("order = %s, %s", listed[0].ID, listed[1].ID)
	}
}

func TestListMonitorIncidentsUnknownMonitor(t *testing.T) {
	handler := newTestHandler()
	path := "/api/monitors/" + uuid.NewString() + "/incidents"
	rec := doRequest(t, handler, http.MethodGet, path, "")
	assertErrorStatus(t, rec, http.StatusNotFound, "monitor not found")
}

func TestGetActiveIncident(t *testing.T) {
	env := newTestEnv()
	monitor := createJSONMonitor(t, env.handler, "https://example.com/health", http.StatusOK)

	open := models.Incident{
		MonitorID:    monitor.ID,
		StartedAt:    time.Now().UTC(),
		Status:       models.IncidentStatusOpen,
		FailureCount: 3,
	}
	if err := env.incidents.Create(nil, &open); err != nil {
		t.Fatal(err)
	}

	rec := doRequest(t, env.handler, http.MethodGet, "/api/monitors/"+monitor.ID.String()+"/incidents/active", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var got models.Incident
	decodeBody(t, rec, &got)
	if got.ID != open.ID || got.Status != models.IncidentStatusOpen {
		t.Fatalf("got = %+v", got)
	}
}

func TestGetActiveIncidentNone(t *testing.T) {
	env := newTestEnv()
	monitor := createJSONMonitor(t, env.handler, "https://example.com/health", http.StatusOK)

	rec := doRequest(t, env.handler, http.MethodGet, "/api/monitors/"+monitor.ID.String()+"/incidents/active", "")
	assertErrorStatus(t, rec, http.StatusNotFound, "no open incident")
}

func TestGetActiveIncidentUnknownMonitor(t *testing.T) {
	handler := newTestHandler()
	path := "/api/monitors/" + uuid.NewString() + "/incidents/active"
	rec := doRequest(t, handler, http.MethodGet, path, "")
	assertErrorStatus(t, rec, http.StatusNotFound, "monitor not found")
}
