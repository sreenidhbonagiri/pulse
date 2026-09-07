package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
)

const validMonitorJSON = `{
	"name": "My API",
	"url": "https://example.com/health",
	"http_method": "GET",
	"check_interval_seconds": 60,
	"timeout_seconds": 5,
	"expected_status_code": 200
}`

func TestCreateMonitorSuccess(t *testing.T) {
	handler := newTestHandler()

	rec := doRequest(t, handler, http.MethodPost, "/api/monitors", validMonitorJSON)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var monitor models.Monitor
	decodeBody(t, rec, &monitor)

	if monitor.ID == uuid.Nil {
		t.Fatal("expected monitor id to be set")
	}
	if monitor.Name != "My API" {
		t.Fatalf("name = %q", monitor.Name)
	}
	if monitor.URL != "https://example.com/health" {
		t.Fatalf("url = %q", monitor.URL)
	}
	if monitor.HTTPMethod != http.MethodGet {
		t.Fatalf("http_method = %q", monitor.HTTPMethod)
	}
	if !monitor.IsActive {
		t.Fatal("expected is_active to default to true")
	}
}

func TestCreateMonitorInvalidURL(t *testing.T) {
	handler := newTestHandler()
	body := `{
		"name": "My API",
		"url": "not-a-url",
		"http_method": "GET",
		"check_interval_seconds": 60,
		"timeout_seconds": 5,
		"expected_status_code": 200
	}`

	rec := doRequest(t, handler, http.MethodPost, "/api/monitors", body)
	assertErrorStatus(t, rec, http.StatusBadRequest, "url is invalid")
}

func TestCreateMonitorInvalidMethod(t *testing.T) {
	handler := newTestHandler()
	body := `{
		"name": "My API",
		"url": "https://example.com/health",
		"http_method": "FETCH",
		"check_interval_seconds": 60,
		"timeout_seconds": 5,
		"expected_status_code": 200
	}`

	rec := doRequest(t, handler, http.MethodPost, "/api/monitors", body)
	assertErrorStatus(t, rec, http.StatusBadRequest, "http_method is invalid")
}

func TestGetMonitorNotFound(t *testing.T) {
	handler := newTestHandler()
	path := "/api/monitors/" + uuid.NewString()

	rec := doRequest(t, handler, http.MethodGet, path, "")
	assertErrorStatus(t, rec, http.StatusNotFound, "monitor not found")
}

func TestUpdateMonitor(t *testing.T) {
	handler := newTestHandler()

	created := doRequest(t, handler, http.MethodPost, "/api/monitors", validMonitorJSON)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", created.Code, created.Body.String())
	}

	var monitor models.Monitor
	decodeBody(t, created, &monitor)

	updateBody := `{
		"name": "Updated API",
		"url": "https://example.com/status",
		"http_method": "HEAD",
		"check_interval_seconds": 30,
		"timeout_seconds": 3,
		"expected_status_code": 204
	}`
	path := "/api/monitors/" + monitor.ID.String()
	updated := doRequest(t, handler, http.MethodPut, path, updateBody)
	if updated.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", updated.Code, updated.Body.String())
	}

	var result models.Monitor
	decodeBody(t, updated, &result)
	if result.Name != "Updated API" {
		t.Fatalf("name = %q", result.Name)
	}
	if result.URL != "https://example.com/status" {
		t.Fatalf("url = %q", result.URL)
	}
	if result.HTTPMethod != http.MethodHead {
		t.Fatalf("http_method = %q", result.HTTPMethod)
	}
	if result.CheckIntervalSeconds != 30 {
		t.Fatalf("check_interval_seconds = %d", result.CheckIntervalSeconds)
	}
}

func TestDeleteMonitor(t *testing.T) {
	handler := newTestHandler()

	created := doRequest(t, handler, http.MethodPost, "/api/monitors", validMonitorJSON)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", created.Code, created.Body.String())
	}

	var monitor models.Monitor
	decodeBody(t, created, &monitor)

	path := "/api/monitors/" + monitor.ID.String()
	deleted := doRequest(t, handler, http.MethodDelete, path, "")
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", deleted.Code, deleted.Body.String())
	}

	got := doRequest(t, handler, http.MethodGet, path, "")
	assertErrorStatus(t, got, http.StatusNotFound, "monitor not found")
}

func newTestHandler() http.Handler {
	return newTestEnv().handler
}

type testEnv struct {
	handler      http.Handler
	checkResults *fakeCheckResultRepo
	incidents    *fakeIncidentRepo
	publisher    *queue.MemoryPublisher
}

func newTestEnv() testEnv {
	monitors := newFakeMonitorRepo()
	checkResults := newFakeCheckResultRepo()
	incidents := newFakeIncidentRepo()
	publisher := queue.NewMemoryPublisher()
	return testEnv{
		handler:      NewServer(config.Config{}, monitors, checkResults, incidents, publisher).Handler(),
		checkResults: checkResults,
		incidents:    incidents,
		publisher:    publisher,
	}
}

func doRequest(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder, dest any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), dest); err != nil {
		t.Fatalf("decode response: %v, body = %s", err, rec.Body.String())
	}
}

func assertErrorStatus(t *testing.T, rec *httptest.ResponseRecorder, status int, message string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, status, rec.Body.String())
	}

	var errBody errorResponse
	decodeBody(t, rec, &errBody)
	if errBody.Error != message {
		t.Fatalf("error = %q, want %q", errBody.Error, message)
	}
}
