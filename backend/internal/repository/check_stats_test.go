package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

func TestPercentileContMatchesExpectedValues(t *testing.T) {
	latencies := []float64{10, 20, 30, 40, 50}
	if got := percentileCont(latencies, 0.50); got != 30 {
		t.Fatalf("p50 = %v, want 30", got)
	}
	if got := round2(percentileCont(latencies, 0.95)); got != 48 {
		t.Fatalf("p95 = %v, want 48", got)
	}
	if got := round2(percentileCont(latencies, 0.99)); got != 49.6 {
		t.Fatalf("p99 = %v, want 49.6", got)
	}
}

func TestComputeCheckStatsIgnoresOldRowsForAggregates(t *testing.T) {
	now := time.Now().UTC()
	since := now.Add(-time.Hour)
	monitorID := uuid.New()

	items := []models.CheckResult{
		{MonitorID: monitorID, Success: false, ResponseTimeMs: 900, CheckedAt: now.Add(-2 * time.Hour)},
		{MonitorID: monitorID, Success: true, ResponseTimeMs: 10, CheckedAt: now.Add(-4 * time.Minute)},
		{MonitorID: monitorID, Success: true, ResponseTimeMs: 20, CheckedAt: now.Add(-3 * time.Minute)},
		{MonitorID: monitorID, Success: false, ResponseTimeMs: 30, CheckedAt: now.Add(-2 * time.Minute)},
		{MonitorID: monitorID, Success: true, ResponseTimeMs: 40, CheckedAt: now.Add(-time.Minute)},
		{MonitorID: monitorID, Success: true, ResponseTimeMs: 50, CheckedAt: now},
	}

	stats := ComputeCheckStats(items, since)
	if stats.LatestSuccess == nil || !*stats.LatestSuccess {
		t.Fatalf("latest = %v, want up", stats.LatestSuccess)
	}
	if stats.TotalChecks != 5 || stats.FailedChecks != 1 {
		t.Fatalf("totals = %d failed=%d", stats.TotalChecks, stats.FailedChecks)
	}
	if stats.AverageLatencyMs != 30 {
		t.Fatalf("avg = %v, want 30", stats.AverageLatencyMs)
	}
	if stats.P50LatencyMs != 30 {
		t.Fatalf("p50 = %v, want 30", stats.P50LatencyMs)
	}
}
