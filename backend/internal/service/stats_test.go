package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/cache"
	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

func TestGetStatsUnknownMonitor(t *testing.T) {
	svc := NewMonitorStatsService(newMemoryMonitors(), newMemoryCheckResults(), newMemoryIncidents(), cache.NewMemory())
	_, err := svc.GetStats(context.Background(), uuid.New())
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestGetStatsCalculations(t *testing.T) {
	monitor := storedMonitor()
	checks := newMemoryCheckResults()
	incidents := newMemoryIncidents()
	now := time.Now().UTC()

	addCheck(t, checks, monitor.ID, true, 10, now.Add(-4*time.Minute))
	addCheck(t, checks, monitor.ID, true, 20, now.Add(-3*time.Minute))
	addCheck(t, checks, monitor.ID, false, 30, now.Add(-2*time.Minute))
	addCheck(t, checks, monitor.ID, true, 40, now.Add(-time.Minute))
	addCheck(t, checks, monitor.ID, true, 50, now)

	_ = incidents.Create(context.Background(), &models.Incident{
		MonitorID:    monitor.ID,
		StartedAt:    now.Add(-time.Hour),
		Status:       models.IncidentStatusResolved,
		FailureCount: 3,
	})
	_ = incidents.Create(context.Background(), &models.Incident{
		MonitorID:    monitor.ID,
		StartedAt:    now.Add(-10 * time.Minute),
		Status:       models.IncidentStatusOpen,
		FailureCount: 3,
	})

	svc := NewMonitorStatsService(newMemoryMonitors(monitor), checks, incidents, cache.NewMemory())
	stats, err := svc.GetStats(context.Background(), monitor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stats.CurrentStatus != models.MonitorStatusUp {
		t.Fatalf("status = %s", stats.CurrentStatus)
	}
	if stats.TotalChecks != 5 || stats.FailedChecks != 1 {
		t.Fatalf("checks = %d failed=%d", stats.TotalChecks, stats.FailedChecks)
	}
	if stats.UptimePercentage != 80 {
		t.Fatalf("uptime = %v, want 80", stats.UptimePercentage)
	}
	if stats.AverageLatencyMs != 30 || stats.P50LatencyMs != 30 {
		t.Fatalf("latency avg=%v p50=%v", stats.AverageLatencyMs, stats.P50LatencyMs)
	}
	if stats.IncidentCount != 2 || !stats.ActiveIncident {
		t.Fatalf("incidents count=%d active=%t", stats.IncidentCount, stats.ActiveIncident)
	}
}

func TestGetStatsCacheHitAndMiss(t *testing.T) {
	monitor := storedMonitor()
	checks := newMemoryCheckResults()
	addCheck(t, checks, monitor.ID, true, 12, time.Now().UTC())
	mem := cache.NewMemory()
	svc := NewMonitorStatsService(newMemoryMonitors(monitor), checks, newMemoryIncidents(), mem)

	first, err := svc.GetStats(context.Background(), monitor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mem.Gets) != 1 || len(mem.Sets) != 1 {
		t.Fatalf("after miss gets=%d sets=%d", len(mem.Gets), len(mem.Sets))
	}

	second, err := svc.GetStats(context.Background(), monitor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mem.Gets) != 2 || len(mem.Sets) != 1 {
		t.Fatalf("after hit gets=%d sets=%d", len(mem.Gets), len(mem.Sets))
	}
	if first.TotalChecks != second.TotalChecks {
		t.Fatalf("cached stats changed")
	}
}

func TestGetStatsRedisFailureFallsBackToPostgres(t *testing.T) {
	monitor := storedMonitor()
	checks := newMemoryCheckResults()
	addCheck(t, checks, monitor.ID, false, 40, time.Now().UTC())
	mem := cache.NewMemory()
	mem.GetErr = errors.New("redis down")
	mem.SetErr = errors.New("redis down")
	svc := NewMonitorStatsService(newMemoryMonitors(monitor), checks, newMemoryIncidents(), mem)

	stats, err := svc.GetStats(context.Background(), monitor.ID)
	if err != nil {
		t.Fatalf("redis failure should not fail the request: %v", err)
	}
	if stats.CurrentStatus != models.MonitorStatusDown || stats.TotalChecks != 1 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestInvalidateCacheAfterNewCheck(t *testing.T) {
	monitor := storedMonitor()
	checks := newMemoryCheckResults()
	mem := cache.NewMemory()
	_ = mem.SetMonitorStats(context.Background(), monitor.ID, models.MonitorStats{TotalChecks: 99}, cache.DefaultTTL)

	svc := NewMonitorCheckService(
		newMemoryMonitors(monitor),
		checks,
		stubChecker{result: models.CheckResult{Success: true, ResponseTimeMs: 15, CheckedAt: time.Now().UTC()}},
		nil,
		nil,
	)
	svc.SetStatsCache(mem)

	if _, err := svc.RunCheck(context.Background(), monitor.ID, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if len(mem.Deletes) != 1 {
		t.Fatalf("deletes = %d, want 1 after new check", len(mem.Deletes))
	}
	got, err := mem.GetMonitorStats(context.Background(), monitor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expected stats cache to be empty after a new check")
	}
}

func TestInvalidateCacheAfterIncidentChange(t *testing.T) {
	h := newIncidentHarness()
	mem := cache.NewMemory()
	_ = mem.SetMonitorStats(context.Background(), h.monitor.ID, models.MonitorStats{TotalChecks: 99}, cache.DefaultTTL)
	h.svc.SetStatsCache(mem)

	h.save(t, false)
	h.save(t, false)
	h.save(t, false)
	if err := h.svc.Evaluate(context.Background(), h.latest(t)); err != nil {
		t.Fatal(err)
	}
	if len(mem.Deletes) != 1 {
		t.Fatalf("deletes = %d, want 1 after opening an incident", len(mem.Deletes))
	}
}

func addCheck(t *testing.T, repo *memoryCheckResults, monitorID uuid.UUID, success bool, latency int, at time.Time) {
	t.Helper()
	result := models.CheckResult{
		JobID:          uuid.New(),
		MonitorID:      monitorID,
		Success:        success,
		ResponseTimeMs: latency,
		CheckedAt:      at,
	}
	if err := repo.Create(context.Background(), &result); err != nil {
		t.Fatal(err)
	}
}
