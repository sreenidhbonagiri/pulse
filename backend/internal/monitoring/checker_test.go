package monitoring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

func TestCheckHTTP200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	result := NewChecker(nil).Check(context.Background(), testMonitor(server.URL, http.MethodGet, 2, http.StatusOK))

	if !result.Success {
		t.Fatalf("success = false, error = %v", strValue(result.ErrorMessage))
	}
	assertStatus(t, result, http.StatusOK)
	if result.ErrorMessage != nil {
		t.Fatalf("error_message = %q, want nil", *result.ErrorMessage)
	}
	if result.ResponseTimeMs < 0 {
		t.Fatalf("response_time_ms = %d", result.ResponseTimeMs)
	}
}

func TestCheckHTTP500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	result := NewChecker(nil).Check(context.Background(), testMonitor(server.URL, http.MethodGet, 2, http.StatusOK))

	if result.Success {
		t.Fatal("expected success = false for HTTP 500")
	}
	assertStatus(t, result, http.StatusInternalServerError)
	assertErrorContains(t, result, "unexpected status code")
}

func TestCheckTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timer := time.NewTimer(3 * time.Second)
		defer timer.Stop()

		select {
		case <-r.Context().Done():
			return
		case <-timer.C:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	result := NewChecker(nil).Check(context.Background(), testMonitor(server.URL, http.MethodGet, 1, http.StatusOK))

	if result.Success {
		t.Fatal("expected success = false for timeout")
	}
	if result.StatusCode != nil {
		t.Fatalf("status_code = %d, want nil on timeout", *result.StatusCode)
	}
	assertErrorContains(t, result, "timed out")
	if result.ResponseTimeMs < 900 {
		t.Fatalf("response_time_ms = %d, want at least ~1000ms timeout", result.ResponseTimeMs)
	}
}

func TestCheckUnexpectedStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	result := NewChecker(nil).Check(context.Background(), testMonitor(server.URL, http.MethodGet, 2, http.StatusCreated))

	if result.Success {
		t.Fatal("expected success = false for unexpected status code")
	}
	assertStatus(t, result, http.StatusOK)
	assertErrorContains(t, result, "got 200, want 201")
}

func TestCheckSlowResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	result := NewChecker(nil).Check(context.Background(), testMonitor(server.URL, http.MethodGet, 2, http.StatusOK))

	if !result.Success {
		t.Fatalf("success = false, error = %v", strValue(result.ErrorMessage))
	}
	assertStatus(t, result, http.StatusOK)
	if result.ResponseTimeMs < 180 {
		t.Fatalf("response_time_ms = %d, want at least 180ms for a slow response", result.ResponseTimeMs)
	}
}

func testMonitor(rawURL, method string, timeoutSeconds, expectedStatus int) models.Monitor {
	return models.Monitor{
		ID:                 uuid.New(),
		Name:               "test",
		URL:                rawURL,
		HTTPMethod:         method,
		TimeoutSeconds:     timeoutSeconds,
		ExpectedStatusCode: expectedStatus,
		IsActive:           true,
	}
}

func assertStatus(t *testing.T, result models.CheckResult, want int) {
	t.Helper()
	if result.StatusCode == nil {
		t.Fatal("status_code is nil")
	}
	if *result.StatusCode != want {
		t.Fatalf("status_code = %d, want %d", *result.StatusCode, want)
	}
}

func assertErrorContains(t *testing.T, result models.CheckResult, want string) {
	t.Helper()
	if result.ErrorMessage == nil {
		t.Fatal("error_message is nil")
	}
	if !strings.Contains(*result.ErrorMessage, want) {
		t.Fatalf("error_message = %q, want substring %q", *result.ErrorMessage, want)
	}
}

func strValue(value *string) string {
	if value == nil {
		return "<nil>"
	}
	return *value
}
