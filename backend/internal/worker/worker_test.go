package worker

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
	"github.com/sreenidhbonagiri/pulse/backend/internal/service"
)

func TestHandleJobSavesCheckResult(t *testing.T) {
	monitor := models.Monitor{
		ID:                 uuid.New(),
		Name:               "test",
		URL:                "https://example.com/health",
		HTTPMethod:         "GET",
		TimeoutSeconds:     2,
		ExpectedStatusCode: 200,
		IsActive:           true,
	}
	status := 200
	results := newMemoryCheckResults()
	svc := service.NewMonitorCheckService(
		newMemoryMonitors(monitor),
		results,
		stubChecker{result: models.CheckResult{
			StatusCode:     &status,
			ResponseTimeMs: 15,
			Success:        true,
			CheckedAt:      time.Now().UTC(),
		}},
		nil,
	)

	job := queue.NewMonitorCheckJob(monitor.ID)
	if err := New(svc).HandleJob(context.Background(), job); err != nil {
		t.Fatalf("HandleJob: %v", err)
	}

	listed, err := results.ListByMonitorID(context.Background(), monitor.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || !listed[0].Success {
		t.Fatalf("saved results = %+v", listed)
	}
}

func TestHandleJobFailedHTTPStatusStillSaves(t *testing.T) {
	monitor := models.Monitor{
		ID:                 uuid.New(),
		Name:               "test",
		URL:                "https://example.com/health",
		HTTPMethod:         "GET",
		TimeoutSeconds:     2,
		ExpectedStatusCode: 200,
		IsActive:           true,
	}
	status := 500
	message := "unexpected status code"
	results := newMemoryCheckResults()
	svc := service.NewMonitorCheckService(
		newMemoryMonitors(monitor),
		results,
		stubChecker{result: models.CheckResult{
			StatusCode:     &status,
			ResponseTimeMs: 20,
			Success:        false,
			ErrorMessage:   &message,
			CheckedAt:      time.Now().UTC(),
		}},
		nil,
	)

	if err := New(svc).HandleJob(context.Background(), queue.NewMonitorCheckJob(monitor.ID)); err != nil {
		t.Fatalf("HandleJob: %v", err)
	}

	listed, err := results.ListByMonitorID(context.Background(), monitor.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Success || listed[0].StatusCode == nil || *listed[0].StatusCode != 500 {
		t.Fatalf("saved results = %+v", listed)
	}
}

func TestHandleJobUnknownMonitorDoesNotFail(t *testing.T) {
	svc := service.NewMonitorCheckService(newMemoryMonitors(), newMemoryCheckResults(), stubChecker{}, nil)
	err := New(svc).HandleJob(context.Background(), queue.NewMonitorCheckJob(uuid.New()))
	if err != nil {
		t.Fatalf("HandleJob: %v, want nil so the message can be acked", err)
	}
}

type stubChecker struct {
	result models.CheckResult
}

func (s stubChecker) Check(_ context.Context, monitor models.Monitor) models.CheckResult {
	result := s.result
	result.MonitorID = monitor.ID
	return result
}

type memoryMonitors struct {
	monitors map[uuid.UUID]models.Monitor
}

func newMemoryMonitors(items ...models.Monitor) *memoryMonitors {
	repo := &memoryMonitors{monitors: make(map[uuid.UUID]models.Monitor)}
	for _, item := range items {
		repo.monitors[item.ID] = item
	}
	return repo
}

func (m *memoryMonitors) Create(_ context.Context, monitor *models.Monitor) error {
	m.monitors[monitor.ID] = *monitor
	return nil
}

func (m *memoryMonitors) GetByID(_ context.Context, id uuid.UUID) (*models.Monitor, error) {
	monitor, ok := m.monitors[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copied := monitor
	return &copied, nil
}

func (m *memoryMonitors) List(_ context.Context) ([]models.Monitor, error) { return nil, nil }

func (m *memoryMonitors) Update(_ context.Context, monitor *models.Monitor) error {
	m.monitors[monitor.ID] = *monitor
	return nil
}

func (m *memoryMonitors) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.monitors, id)
	return nil
}

type memoryCheckResults struct {
	results map[uuid.UUID]models.CheckResult
}

func newMemoryCheckResults() *memoryCheckResults {
	return &memoryCheckResults{results: make(map[uuid.UUID]models.CheckResult)}
}

func (m *memoryCheckResults) Create(_ context.Context, result *models.CheckResult) error {
	if result.ID == uuid.Nil {
		result.ID = uuid.New()
	}
	m.results[result.ID] = *result
	return nil
}

func (m *memoryCheckResults) GetByID(_ context.Context, id uuid.UUID) (*models.CheckResult, error) {
	result, ok := m.results[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copied := result
	return &copied, nil
}

func (m *memoryCheckResults) ListByMonitorID(_ context.Context, monitorID uuid.UUID, limit int) ([]models.CheckResult, error) {
	listed := make([]models.CheckResult, 0)
	for _, result := range m.results {
		if result.MonitorID == monitorID {
			listed = append(listed, result)
		}
	}
	if limit <= 0 || limit > len(listed) {
		return listed, nil
	}
	return listed[:limit], nil
}
