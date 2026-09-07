package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
	"github.com/sreenidhbonagiri/pulse/backend/internal/database"
	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

func TestPostgresCheckResultRepository(t *testing.T) {
	ctx := context.Background()
	pool, err := database.Connect(ctx, config.Load().DatabaseURL)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	monitors := NewPostgresMonitorRepository(pool)
	results := NewPostgresCheckResultRepository(pool)
	monitor := createRepoTestMonitor(t, ctx, monitors)

	t.Run("create and get by id", func(t *testing.T) {
		status := 200
		saved := &models.CheckResult{
			MonitorID:      monitor.ID,
			StatusCode:     &status,
			ResponseTimeMs: 42,
			Success:        true,
			CheckedAt:      time.Now().UTC().Add(-2 * time.Minute),
		}
		if err := results.Create(ctx, saved); err != nil {
			t.Fatalf("create: %v", err)
		}
		if saved.ID == uuid.Nil {
			t.Fatal("expected id to be set")
		}

		got, err := results.GetByID(ctx, saved.ID)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.MonitorID != monitor.ID {
			t.Fatalf("monitor_id = %s, want %s", got.MonitorID, monitor.ID)
		}
		if got.StatusCode == nil || *got.StatusCode != 200 {
			t.Fatalf("status_code = %v", got.StatusCode)
		}
		if !got.Success {
			t.Fatal("expected success = true")
		}
	})

	t.Run("duplicate job_id is rejected", func(t *testing.T) {
		jobID := uuid.New()
		status := 200
		first := &models.CheckResult{
			JobID:          jobID,
			MonitorID:      monitor.ID,
			StatusCode:     &status,
			ResponseTimeMs: 10,
			Success:        true,
			CheckedAt:      time.Now().UTC(),
		}
		if err := results.Create(ctx, first); err != nil {
			t.Fatalf("create: %v", err)
		}

		second := &models.CheckResult{
			JobID:          jobID,
			MonitorID:      monitor.ID,
			StatusCode:     &status,
			ResponseTimeMs: 11,
			Success:        true,
			CheckedAt:      time.Now().UTC(),
		}
		err := results.Create(ctx, second)
		if !errors.Is(err, ErrDuplicate) {
			t.Fatalf("err = %v, want ErrDuplicate", err)
		}

		listed, err := results.ListByMonitorID(ctx, monitor.ID, 100)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		matches := 0
		for _, item := range listed {
			if item.JobID == jobID {
				matches++
			}
		}
		if matches != 1 {
			t.Fatalf("job_id rows = %d, want 1", matches)
		}
	})

	t.Run("get missing result", func(t *testing.T) {
		_, err := results.GetByID(ctx, uuid.New())
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("list newest first with limit", func(t *testing.T) {
		monitor := createRepoTestMonitor(t, ctx, monitors)
		older := createRepoTestResult(t, ctx, results, monitor.ID, 100, time.Now().UTC().Add(-3*time.Minute))
		middle := createRepoTestResult(t, ctx, results, monitor.ID, 200, time.Now().UTC().Add(-2*time.Minute))
		newest := createRepoTestResult(t, ctx, results, monitor.ID, 300, time.Now().UTC().Add(-1*time.Minute))
		_ = older

		listed, err := results.ListByMonitorID(ctx, monitor.ID, 2)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(listed) != 2 {
			t.Fatalf("len = %d, want 2", len(listed))
		}
		if listed[0].ID != newest.ID || listed[1].ID != middle.ID {
			t.Fatalf("order = %s, %s; want newest then middle", listed[0].ID, listed[1].ID)
		}

		all, err := results.ListByMonitorID(ctx, monitor.ID, 50)
		if err != nil {
			t.Fatalf("list all: %v", err)
		}
		if len(all) != 3 {
			t.Fatalf("len = %d, want 3", len(all))
		}
	})

	t.Run("nullable status and error", func(t *testing.T) {
		message := "request timed out"
		saved := &models.CheckResult{
			MonitorID:      monitor.ID,
			StatusCode:     nil,
			ResponseTimeMs: 1000,
			Success:        false,
			ErrorMessage:   &message,
			CheckedAt:      time.Now().UTC(),
		}
		if err := results.Create(ctx, saved); err != nil {
			t.Fatalf("create: %v", err)
		}

		got, err := results.GetByID(ctx, saved.ID)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.StatusCode != nil {
			t.Fatalf("status_code = %d, want nil", *got.StatusCode)
		}
		if got.ErrorMessage == nil || *got.ErrorMessage != message {
			t.Fatalf("error_message = %v", got.ErrorMessage)
		}
	})

	t.Run("delete monitor cascades results", func(t *testing.T) {
		orphanMonitor := createRepoTestMonitor(t, ctx, monitors)
		saved := createRepoTestResult(t, ctx, results, orphanMonitor.ID, 200, time.Now().UTC())

		if err := monitors.Delete(ctx, orphanMonitor.ID); err != nil {
			t.Fatalf("delete monitor: %v", err)
		}

		_, err := results.GetByID(ctx, saved.ID)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound after cascade", err)
		}
	})
}

func createRepoTestMonitor(t *testing.T, ctx context.Context, monitors *PostgresMonitorRepository) *models.Monitor {
	t.Helper()
	monitor := &models.Monitor{
		Name:                 "check-result-test-" + uuid.NewString(),
		URL:                  "https://example.com/health",
		HTTPMethod:           "GET",
		CheckIntervalSeconds: 60,
		TimeoutSeconds:       5,
		ExpectedStatusCode:   200,
		IsActive:             true,
	}
	if err := monitors.Create(ctx, monitor); err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	t.Cleanup(func() {
		_ = monitors.Delete(ctx, monitor.ID)
	})
	return monitor
}

func createRepoTestResult(t *testing.T, ctx context.Context, results *PostgresCheckResultRepository, monitorID uuid.UUID, status int, checkedAt time.Time) *models.CheckResult {
	t.Helper()
	result := &models.CheckResult{
		MonitorID:      monitorID,
		StatusCode:     &status,
		ResponseTimeMs: status,
		Success:        status == 200,
		CheckedAt:      checkedAt,
	}
	if err := results.Create(ctx, result); err != nil {
		t.Fatalf("create result: %v", err)
	}
	return result
}
