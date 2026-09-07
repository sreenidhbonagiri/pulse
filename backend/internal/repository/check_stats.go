package repository

import (
	"math"
	"sort"
	"time"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

// CheckStats is the PostgreSQL aggregation used to build dashboard stats.
type CheckStats struct {
	TotalChecks      int
	FailedChecks     int
	AverageLatencyMs float64
	P50LatencyMs     float64
	P95LatencyMs     float64
	P99LatencyMs     float64
	LatestSuccess    *bool
}

// ComputeCheckStats calculates the same aggregates as the PostgreSQL query.
// Tests and in-memory fakes use this so they stay consistent with percentile_cont.
func ComputeCheckStats(results []models.CheckResult, since time.Time) CheckStats {
	var stats CheckStats
	if len(results) == 0 {
		return stats
	}

	latest := results[0]
	for _, result := range results {
		if result.CheckedAt.After(latest.CheckedAt) {
			latest = result
		}
	}
	success := latest.Success
	stats.LatestSuccess = &success

	latencies := make([]float64, 0)
	for _, result := range results {
		if result.CheckedAt.Before(since) {
			continue
		}
		stats.TotalChecks++
		if !result.Success {
			stats.FailedChecks++
		}
		latencies = append(latencies, float64(result.ResponseTimeMs))
	}
	if stats.TotalChecks == 0 {
		return stats
	}

	sort.Float64s(latencies)
	sum := 0.0
	for _, latency := range latencies {
		sum += latency
	}
	stats.AverageLatencyMs = round2(sum / float64(len(latencies)))
	stats.P50LatencyMs = round2(percentileCont(latencies, 0.50))
	stats.P95LatencyMs = round2(percentileCont(latencies, 0.95))
	stats.P99LatencyMs = round2(percentileCont(latencies, 0.99))
	return stats
}

func percentileCont(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	pos := p * float64(len(sorted)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sorted[lo]
	}
	weight := pos - float64(lo)
	return sorted[lo]*(1-weight) + sorted[hi]*weight
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
