package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

func TestRunCheckSuccess(t *testing.T) {
	monitor := storedMonitor()
	status := 200
	checker := stubChecker{result: models.CheckResult{
		StatusCode:     &status,
		ResponseTimeMs: 12,
		Success:        true,
		CheckedAt:      time.Now().UTC(),
	}}
	results := newMemoryCheckResults()
	svc := NewMonitorCheckService(newMemoryMonitors(monitor), results, checker, nil)

	got, err := svc.RunCheck(context.Background(), monitor.ID)
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if !got.Success {
		t.Fatal("expected success")
	}
	if got.MonitorID != monitor.ID {
		t.Fatalf("monitor_id = %s", got.MonitorID)
	}
	if got.ID == uuid.Nil {
		t.Fatal("expected saved result id")
	}
	if _, err := results.GetByID(context.Background(), got.ID); err != nil {
		t.Fatalf("result was not saved: %v", err)
	}
}

func TestRunCheckUnknownMonitor(t *testing.T) {
	svc := NewMonitorCheckService(newMemoryMonitors(), newMemoryCheckResults(), stubChecker{}, nil)

	_, err := svc.RunCheck(context.Background(), uuid.New())
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestListChecksNewestFirst(t *testing.T) {
	monitor := storedMonitor()
	results := newMemoryCheckResults()
	svc := NewMonitorCheckService(newMemoryMonitors(monitor), results, stubChecker{}, nil)

	older := models.CheckResult{MonitorID: monitor.ID, ResponseTimeMs: 1, CheckedAt: time.Now().UTC().Add(-time.Minute)}
	newer := models.CheckResult{MonitorID: monitor.ID, ResponseTimeMs: 2, CheckedAt: time.Now().UTC()}
	_ = results.Create(context.Background(), &older)
	_ = results.Create(context.Background(), &newer)

	listed, err := svc.ListChecks(context.Background(), monitor.ID, 10)
	if err != nil {
		t.Fatalf("ListChecks: %v", err)
	}
	if len(listed) != 2 || listed[0].ID != newer.ID || listed[1].ID != older.ID {
		t.Fatalf("unexpected list order: %+v", listed)
	}
}

func TestListChecksUnknownMonitor(t *testing.T) {
	svc := NewMonitorCheckService(newMemoryMonitors(), newMemoryCheckResults(), stubChecker{}, nil)

	_, err := svc.ListChecks(context.Background(), uuid.New(), 10)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestEnqueueCheck(t *testing.T) {
	monitor := storedMonitor()
	publisher := queue.NewMemoryPublisher()
	svc := NewMonitorCheckService(newMemoryMonitors(monitor), newMemoryCheckResults(), nil, publisher)

	job, err := svc.EnqueueCheck(context.Background(), monitor.ID)
	if err != nil {
		t.Fatalf("EnqueueCheck: %v", err)
	}
	if job.JobID == uuid.Nil || job.MonitorID != monitor.ID {
		t.Fatalf("job = %+v", job)
	}
	if len(publisher.Jobs) != 1 || publisher.Jobs[0].JobID != job.JobID {
		t.Fatalf("published jobs = %+v", publisher.Jobs)
	}
}

func TestEnqueueCheckUnknownMonitor(t *testing.T) {
	publisher := queue.NewMemoryPublisher()
	svc := NewMonitorCheckService(newMemoryMonitors(), newMemoryCheckResults(), nil, publisher)

	_, err := svc.EnqueueCheck(context.Background(), uuid.New())
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if len(publisher.Jobs) != 0 {
		t.Fatal("should not publish a job for an unknown monitor")
	}
}

type stubChecker struct {
	result models.CheckResult
}

func (s stubChecker) Check(_ context.Context, monitor models.Monitor) models.CheckResult {
	result := s.result
	result.MonitorID = monitor.ID
	if result.CheckedAt.IsZero() {
		result.CheckedAt = time.Now().UTC()
	}
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

func (m *memoryMonitors) List(_ context.Context) ([]models.Monitor, error) {
	return nil, nil
}

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
	for i := 0; i < len(listed); i++ {
		for j := i + 1; j < len(listed); j++ {
			if listed[j].CheckedAt.After(listed[i].CheckedAt) {
				listed[i], listed[j] = listed[j], listed[i]
			}
		}
	}
	if limit <= 0 || limit > len(listed) {
		return listed, nil
	}
	return listed[:limit], nil
}

func storedMonitor() models.Monitor {
	return models.Monitor{
		ID:                 uuid.New(),
		Name:               "test",
		URL:                "https://example.com/health",
		HTTPMethod:         "GET",
		TimeoutSeconds:     2,
		ExpectedStatusCode: 200,
		IsActive:           true,
	}
}
