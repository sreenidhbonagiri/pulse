package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/cache"
	"github.com/sreenidhbonagiri/pulse/backend/internal/metrics"
	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

const (
	consecutiveFailuresToOpen     = 3
	consecutiveSuccessesToResolve = 2
)

// IncidentService turns saved CheckResults into open and resolved incidents.
type IncidentService struct {
	checks     repository.CheckResultRepository
	incidents  repository.IncidentRepository
	transactor repository.Transactor
	statsCache cache.MonitorStatsCache
}

func NewIncidentService(
	checks repository.CheckResultRepository,
	incidents repository.IncidentRepository,
	transactor repository.Transactor,
) *IncidentService {
	return &IncidentService{
		checks:     checks,
		incidents:  incidents,
		transactor: transactor,
	}
}

func (s *IncidentService) SetStatsCache(statsCache cache.MonitorStatsCache) {
	s.statsCache = statsCache
}

func (s *IncidentService) Evaluate(ctx context.Context, result models.CheckResult) error {
	if s == nil {
		return nil
	}

	err := s.transactor.WithMonitorLock(ctx, result.MonitorID, func(ctx context.Context) error {
		return s.evaluateLocked(ctx, result.MonitorID)
	})
	if errors.Is(err, repository.ErrNotFound) {
		return nil
	}
	return err
}

func (s *IncidentService) ListByMonitorID(ctx context.Context, monitorID uuid.UUID, limit int) ([]models.Incident, error) {
	return s.incidents.ListByMonitorID(ctx, monitorID, limit)
}

func (s *IncidentService) GetOpenByMonitorID(ctx context.Context, monitorID uuid.UUID) (*models.Incident, error) {
	return s.incidents.GetOpenByMonitorID(ctx, monitorID)
}

func (s *IncidentService) evaluateLocked(ctx context.Context, monitorID uuid.UUID) error {
	checks, err := s.checks.ListByMonitorID(ctx, monitorID, 100)
	if err != nil {
		return err
	}

	open, err := s.incidents.GetOpenByMonitorID(ctx, monitorID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return err
	}
	if errors.Is(err, repository.ErrNotFound) {
		open = nil
	}

	failStreak, okStreak := consecutiveStreaks(checks)

	if open != nil {
		if okStreak >= consecutiveSuccessesToResolve {
			resolved, err := s.incidents.Resolve(ctx, open.ID, time.Now().UTC())
			if errors.Is(err, repository.ErrNotFound) {
				return nil
			}
			if err != nil {
				return err
			}
			log.Printf("incident resolved incident_id=%s monitor_id=%s", resolved.ID, monitorID)
			metrics.Default().DecActiveIncidents()
			cache.InvalidateMonitorStats(ctx, s.statsCache, monitorID)
			return nil
		}

		if failStreak > 0 {
			count := failureCountSince(checks, open.StartedAt)
			if count < open.FailureCount {
				count = open.FailureCount
			}
			if count != open.FailureCount {
				if _, err := s.incidents.IncrementFailureCount(ctx, open.ID, count); err != nil && !errors.Is(err, repository.ErrNotFound) {
					return err
				}
				cache.InvalidateMonitorStats(ctx, s.statsCache, monitorID)
			}
		}
		return nil
	}

	if failStreak < consecutiveFailuresToOpen {
		return nil
	}

	startedAt := checks[failStreak-1].CheckedAt
	incident := models.Incident{
		MonitorID:    monitorID,
		StartedAt:    startedAt,
		Status:       models.IncidentStatusOpen,
		FailureCount: failStreak,
	}
	if err := s.incidents.Create(ctx, &incident); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return s.updateExistingOpenCount(ctx, monitorID, checks)
		}
		return err
	}
	log.Printf("incident opened incident_id=%s monitor_id=%s failure_count=%d", incident.ID, monitorID, incident.FailureCount)
	metrics.Default().IncActiveIncidents()
	cache.InvalidateMonitorStats(ctx, s.statsCache, monitorID)
	return nil
}

func (s *IncidentService) updateExistingOpenCount(ctx context.Context, monitorID uuid.UUID, checks []models.CheckResult) error {
	open, err := s.incidents.GetOpenByMonitorID(ctx, monitorID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	count := failureCountSince(checks, open.StartedAt)
	if count < open.FailureCount {
		return nil
	}
	_, err = s.incidents.IncrementFailureCount(ctx, open.ID, count)
	if errors.Is(err, repository.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	cache.InvalidateMonitorStats(ctx, s.statsCache, monitorID)
	return nil
}

func consecutiveStreaks(checks []models.CheckResult) (failures, successes int) {
	for _, check := range checks {
		if check.Success {
			if failures > 0 {
				break
			}
			successes++
			continue
		}
		if successes > 0 {
			break
		}
		failures++
	}
	return failures, successes
}

func failureCountSince(checks []models.CheckResult, startedAt time.Time) int {
	count := 0
	for _, check := range checks {
		if check.CheckedAt.Before(startedAt) {
			continue
		}
		if !check.Success {
			count++
		}
	}
	return count
}
