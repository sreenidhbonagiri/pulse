package api

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
)

func TestEnqueueCheckAccepted(t *testing.T) {
	env := newTestEnv()
	monitor := createJSONMonitor(t, env.handler, "https://example.com/health", http.StatusOK)

	rec := doRequest(t, env.handler, http.MethodPost, "/api/monitors/"+monitor.ID.String()+"/check", "")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var job queue.MonitorCheckJob
	decodeBody(t, rec, &job)
	if job.JobID == uuid.Nil {
		t.Fatal("expected job_id")
	}
	if job.MonitorID != monitor.ID {
		t.Fatalf("monitor_id = %s, want %s", job.MonitorID, monitor.ID)
	}
	if len(env.publisher.Jobs) != 1 || env.publisher.Jobs[0].JobID != job.JobID {
		t.Fatalf("published jobs = %+v", env.publisher.Jobs)
	}

	listed := doRequest(t, env.handler, http.MethodGet, "/api/monitors/"+monitor.ID.String()+"/checks", "")
	var results []models.CheckResult
	decodeBody(t, listed, &results)
	if len(results) != 0 {
		t.Fatal("enqueue should not run the check immediately")
	}
}

func TestEnqueueCheckUnknownMonitor(t *testing.T) {
	env := newTestEnv()
	path := "/api/monitors/" + uuid.NewString() + "/check"

	rec := doRequest(t, env.handler, http.MethodPost, path, "")
	assertErrorStatus(t, rec, http.StatusNotFound, "monitor not found")
	if len(env.publisher.Jobs) != 0 {
		t.Fatal("should not publish a job for an unknown monitor")
	}
}

func TestListMonitorChecks(t *testing.T) {
	env := newTestEnv()
	monitor := createJSONMonitor(t, env.handler, "https://example.com/health", http.StatusOK)

	older := models.CheckResult{
		MonitorID:      monitor.ID,
		ResponseTimeMs: 10,
		Success:        true,
		CheckedAt:      time.Now().UTC().Add(-time.Minute),
	}
	newer := models.CheckResult{
		MonitorID:      monitor.ID,
		ResponseTimeMs: 20,
		Success:        true,
		CheckedAt:      time.Now().UTC(),
	}
	if err := env.checkResults.Create(nil, &older); err != nil {
		t.Fatal(err)
	}
	if err := env.checkResults.Create(nil, &newer); err != nil {
		t.Fatal(err)
	}

	listed := doRequest(t, env.handler, http.MethodGet, "/api/monitors/"+monitor.ID.String()+"/checks", "")
	if listed.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", listed.Code, listed.Body.String())
	}

	var results []models.CheckResult
	decodeBody(t, listed, &results)
	if len(results) != 2 {
		t.Fatalf("len = %d, want 2", len(results))
	}
	if results[0].ID != newer.ID {
		t.Fatalf("first listed id = %s, want newest %s", results[0].ID, newer.ID)
	}
}

func TestListMonitorChecksUnknownMonitor(t *testing.T) {
	handler := newTestHandler()
	path := "/api/monitors/" + uuid.NewString() + "/checks"

	rec := doRequest(t, handler, http.MethodGet, path, "")
	assertErrorStatus(t, rec, http.StatusNotFound, "monitor not found")
}

func createJSONMonitor(t *testing.T, handler http.Handler, targetURL string, expectedStatus int) models.Monitor {
	t.Helper()

	body := fmt.Sprintf(`{
		"name": "Checked API",
		"url": %q,
		"http_method": "GET",
		"check_interval_seconds": 60,
		"timeout_seconds": 2,
		"expected_status_code": %d
	}`, targetURL, expectedStatus)

	rec := doRequest(t, handler, http.MethodPost, "/api/monitors", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create monitor status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var monitor models.Monitor
	decodeBody(t, rec, &monitor)
	return monitor
}
