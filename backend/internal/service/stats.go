package service

import (
	"context"
	"errors"
	"log"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/cache"
	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

const StatsWindow = 24 * time.Hour

// MonitorStatsService builds dashboard stats with a Redis cache-aside layer.
type MonitorStatsService struct {
	monitors  repository.MonitorRepository
	checks    repository.CheckResultRepository
	incidents repository.IncidentRepository
	cache     cache.MonitorStatsCache
}

func NewMonitorStatsService(
	monitors repository.MonitorRepository,
	checks repository.CheckResultRepository,
	incidents repository.IncidentRepository,
	statsCache cache.MonitorStatsCache,
) *MonitorStatsService {
	return &MonitorStatsService{
		monitors:  monitors,
		checks:    checks,
		incidents: incidents,
		cache:     statsCache,
	}
}

func (s *MonitorStatsService) GetStats(ctx context.Context, monitorID uuid.UUID) (*models.MonitorStats, error) {
	if _, err := s.monitors.GetByID(ctx, monitorID); err != nil {
		return nil, err
	}

	if cached, ok := s.cacheGet(ctx, monitorID); ok {
		return cached, nil
	}

	stats, err := s.compute(ctx, monitorID)
	if err != nil {
		return nil, err
	}

	s.cacheSet(ctx, monitorID, *stats)
	return stats, nil
}

func (s *MonitorStatsService) compute(ctx context.Context, monitorID uuid.UUID) (*models.MonitorStats, error) {
	since := time.Now().UTC().Add(-StatsWindow)

	checkStats, err := s.checks.GetCheckStats(ctx, monitorID, since)
	if err != nil {
		return nil, err
	}

	incidentCount, err := s.incidents.CountByMonitorID(ctx, monitorID, since)
	if err != nil {
		return nil, err
	}

	_, err = s.incidents.GetOpenByMonitorID(ctx, monitorID)
	active := err == nil
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	uptime := 0.0
	if checkStats.TotalChecks > 0 {
		successes := float64(checkStats.TotalChecks - checkStats.FailedChecks)
		uptime = roundUptime(successes / float64(checkStats.TotalChecks) * 100)
	}

	return &models.MonitorStats{
		CurrentStatus:    currentStatus(checkStats.LatestSuccess),
		UptimePercentage: uptime,
		AverageLatencyMs: checkStats.AverageLatencyMs,
		P50LatencyMs:     checkStats.P50LatencyMs,
		P95LatencyMs:     checkStats.P95LatencyMs,
		P99LatencyMs:     checkStats.P99LatencyMs,
		TotalChecks:      checkStats.TotalChecks,
		FailedChecks:     checkStats.FailedChecks,
		IncidentCount:    incidentCount,
		ActiveIncident:   active,
	}, nil
}

func (s *MonitorStatsService) cacheGet(ctx context.Context, monitorID uuid.UUID) (*models.MonitorStats, bool) {
	if s.cache == nil {
		return nil, false
	}
	cached, err := s.cache.GetMonitorStats(ctx, monitorID)
	if err != nil {
		log.Printf("stats cache get monitor_id=%s: %v", monitorID, err)
		return nil, false
	}
	if cached == nil {
		return nil, false
	}
	return cached, true
}

func (s *MonitorStatsService) cacheSet(ctx context.Context, monitorID uuid.UUID, stats models.MonitorStats) {
	if s.cache == nil {
		return
	}
	if err := s.cache.SetMonitorStats(ctx, monitorID, stats, cache.DefaultTTL); err != nil {
		log.Printf("stats cache set monitor_id=%s: %v", monitorID, err)
	}
}

func currentStatus(latest *bool) string {
	if latest == nil {
		return models.MonitorStatusUnknown
	}
	if *latest {
		return models.MonitorStatusUp
	}
	return models.MonitorStatusDown
}

func roundUptime(value float64) float64 {
	return math.Round(value*100) / 100
}
