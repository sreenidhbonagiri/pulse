package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

func TestManualCheckSuccess(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	handler := newTestHandler()
	monitor := createJSONMonitor(t, handler, target.URL, http.StatusOK)

	rec := doRequest(t, handler, http.MethodPost, "/api/monitors/"+monitor.ID.String()+"/check", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var result models.CheckResult
	decodeBody(t, rec, &result)
	if result.ID == uuid.Nil {
		t.Fatal("expected saved check result id")
	}
	if result.MonitorID != monitor.ID {
		t.Fatalf("monitor_id = %s, want %s", result.MonitorID, monitor.ID)
	}
	if !result.Success {
		t.Fatalf("success = false, error = %v", result.ErrorMessage)
	}
	if result.StatusCode == nil || *result.StatusCode != http.StatusOK {
		t.Fatalf("status_code = %v", result.StatusCode)
	}
}

func TestManualCheckFailedHTTPStatus(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer target.Close()

	handler := newTestHandler()
	monitor := createJSONMonitor(t, handler, target.URL, http.StatusOK)

	rec := doRequest(t, handler, http.MethodPost, "/api/monitors/"+monitor.ID.String()+"/check", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var result models.CheckResult
	decodeBody(t, rec, &result)
	if result.Success {
		t.Fatal("expected success = false when the target returns 500")
	}
	if result.StatusCode == nil || *result.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status_code = %v", result.StatusCode)
	}
	if result.ErrorMessage == nil {
		t.Fatal("expected error_message")
	}
}

func TestManualCheckUnknownMonitor(t *testing.T) {
	handler := newTestHandler()
	path := "/api/monitors/" + uuid.NewString() + "/check"

	rec := doRequest(t, handler, http.MethodPost, path, "")
	assertErrorStatus(t, rec, http.StatusNotFound, "monitor not found")
}

func TestManualCheckReturnsSavedResult(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	handler := newTestHandler()
	monitor := createJSONMonitor(t, handler, target.URL, http.StatusOK)

	rec := doRequest(t, handler, http.MethodPost, "/api/monitors/"+monitor.ID.String()+"/check", "")
	var created models.CheckResult
	decodeBody(t, rec, &created)

	listed := doRequest(t, handler, http.MethodGet, "/api/monitors/"+monitor.ID.String()+"/checks", "")
	if listed.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listed.Code, listed.Body.String())
	}

	var results []models.CheckResult
	decodeBody(t, listed, &results)
	if len(results) != 1 {
		t.Fatalf("len = %d, want 1", len(results))
	}
	if results[0].ID != created.ID {
		t.Fatalf("listed id = %s, want %s", results[0].ID, created.ID)
	}
}

func TestListMonitorChecks(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	handler := newTestHandler()
	monitor := createJSONMonitor(t, handler, target.URL, http.StatusOK)
	path := "/api/monitors/" + monitor.ID.String() + "/check"

	first := doRequest(t, handler, http.MethodPost, path, "")
	time.Sleep(2 * time.Millisecond)
	second := doRequest(t, handler, http.MethodPost, path, "")
	if first.Code != http.StatusCreated || second.Code != http.StatusCreated {
		t.Fatalf("check statuses = %d, %d", first.Code, second.Code)
	}

	var newer models.CheckResult
	decodeBody(t, second, &newer)

	listed := doRequest(t, handler, http.MethodGet, "/api/monitors/"+monitor.ID.String()+"/checks", "")
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
