package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/cache"
	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

// HTTPChecker runs one HTTP check. monitoring.Checker implements this.
type HTTPChecker interface {
	Check(ctx context.Context, monitor models.Monitor) models.CheckResult
}

// MonitorCheckService coordinates monitors, the queue, and saved check results.
type MonitorCheckService struct {
	monitors     repository.MonitorRepository
	checkResults repository.CheckResultRepository
	checker      HTTPChecker
	publisher    queue.Publisher
	incidents    *IncidentService
	statsCache   cache.MonitorStatsCache
}

func NewMonitorCheckService(
	monitors repository.MonitorRepository,
	checkResults repository.CheckResultRepository,
	checker HTTPChecker,
	publisher queue.Publisher,
	incidents *IncidentService,
) *MonitorCheckService {
	return &MonitorCheckService{
		monitors:     monitors,
		checkResults: checkResults,
		checker:      checker,
		publisher:    publisher,
		incidents:    incidents,
		statsCache:   nil,
	}
}

func (s *MonitorCheckService) SetStatsCache(statsCache cache.MonitorStatsCache) {
	s.statsCache = statsCache
}

func (s *MonitorCheckService) EnqueueCheck(ctx context.Context, monitorID uuid.UUID) (*queue.MonitorCheckJob, error) {
	if _, err := s.monitors.GetByID(ctx, monitorID); err != nil {
		return nil, err
	}

	job := queue.NewMonitorCheckJob(monitorID)
	if err := s.publisher.Publish(ctx, job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *MonitorCheckService) RunCheck(ctx context.Context, monitorID, jobID uuid.UUID) (*models.CheckResult, error) {
	monitor, err := s.monitors.GetByID(ctx, monitorID)
	if err != nil {
		return nil, err
	}

	result := s.checker.Check(ctx, *monitor)
	result.JobID = jobID
	err = s.checkResults.Create(ctx, &result)
	created := true
	if errors.Is(err, repository.ErrDuplicate) {
		saved, lookupErr := s.checkResults.GetByJobID(ctx, jobID)
		if lookupErr != nil {
			return nil, lookupErr
		}
		result = *saved
		created = false
	} else if err != nil {
		return nil, err
	}

	if err := s.incidents.Evaluate(ctx, result); err != nil {
		return nil, err
	}

	if created {
		cache.InvalidateMonitorStats(ctx, s.statsCache, monitorID)
	}

	return &result, nil
}

func (s *MonitorCheckService) ListChecks(ctx context.Context, monitorID uuid.UUID, limit int) ([]models.CheckResult, error) {
	if _, err := s.monitors.GetByID(ctx, monitorID); err != nil {
		return nil, err
	}

	return s.checkResults.ListByMonitorID(ctx, monitorID, limit)
}
