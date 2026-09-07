package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

// HTTPChecker runs one HTTP check. monitoring.Checker implements this.
type HTTPChecker interface {
	Check(ctx context.Context, monitor models.Monitor) models.CheckResult
}

// MonitorCheckService loads a monitor, runs a check, and saves the result.
type MonitorCheckService struct {
	monitors     repository.MonitorRepository
	checkResults repository.CheckResultRepository
	checker      HTTPChecker
}

func NewMonitorCheckService(
	monitors repository.MonitorRepository,
	checkResults repository.CheckResultRepository,
	checker HTTPChecker,
) *MonitorCheckService {
	return &MonitorCheckService{
		monitors:     monitors,
		checkResults: checkResults,
		checker:      checker,
	}
}

func (s *MonitorCheckService) RunCheck(ctx context.Context, monitorID uuid.UUID) (*models.CheckResult, error) {
	monitor, err := s.monitors.GetByID(ctx, monitorID)
	if err != nil {
		return nil, err
	}

	result := s.checker.Check(ctx, *monitor)
	if err := s.checkResults.Create(ctx, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *MonitorCheckService) ListChecks(ctx context.Context, monitorID uuid.UUID, limit int) ([]models.CheckResult, error) {
	if _, err := s.monitors.GetByID(ctx, monitorID); err != nil {
		return nil, err
	}

	return s.checkResults.ListByMonitorID(ctx, monitorID, limit)
}
