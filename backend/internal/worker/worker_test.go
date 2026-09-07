package worker

import (
	"context"
	"errors"
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
	if listed[0].JobID != job.JobID {
		t.Fatalf("job_id = %s, want %s", listed[0].JobID, job.JobID)
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

func TestHandleJobDuplicateDeliveryAcksWithoutSecondRow(t *testing.T) {
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
	worker := New(svc)
	if err := worker.HandleJob(context.Background(), job); err != nil {
		t.Fatalf("first HandleJob: %v", err)
	}
	if err := worker.HandleJob(context.Background(), job); err != nil {
		t.Fatalf("duplicate HandleJob: %v, want nil so the message can be acked", err)
	}

	listed, err := results.ListByMonitorID(context.Background(), monitor.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("len = %d, want 1 CheckResult after duplicate delivery", len(listed))
	}
}

func TestHandleJobInternalFailureIsRetryable(t *testing.T) {
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
	svc := service.NewMonitorCheckService(
		newMemoryMonitors(monitor),
		failingCheckResults{err: errors.New("postgres unavailable")},
		stubChecker{result: models.CheckResult{
			StatusCode:     &status,
			ResponseTimeMs: 15,
			Success:        true,
			CheckedAt:      time.Now().UTC(),
		}},
		nil,
	)

	err := New(svc).HandleJob(context.Background(), queue.NewMonitorCheckJob(monitor.ID))
	if err == nil {
		t.Fatal("expected an internal error so the job can be retried")
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

func (m *memoryMonitors) ListDue(_ context.Context, _ time.Time, _ int) ([]models.Monitor, error) {
	return nil, nil
}

func (m *memoryMonitors) ClaimDue(_ context.Context, _ time.Time, _ func(models.Monitor) error) (*models.Monitor, error) {
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
	if result.JobID != uuid.Nil {
		for _, existing := range m.results {
			if existing.JobID == result.JobID {
				return repository.ErrDuplicate
			}
		}
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

type failingCheckResults struct {
	err error
}

func (f failingCheckResults) Create(_ context.Context, _ *models.CheckResult) error {
	return f.err
}

func (f failingCheckResults) GetByID(_ context.Context, _ uuid.UUID) (*models.CheckResult, error) {
	return nil, repository.ErrNotFound
}

func (f failingCheckResults) ListByMonitorID(_ context.Context, _ uuid.UUID, _ int) ([]models.CheckResult, error) {
	return nil, nil
}
