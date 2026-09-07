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
	svc := service.NewMonitorCheckService(newMemoryMonitors(), newMemoryCheckResults(), stubChecker{}, nil, nil)
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
		nil,
	)

	err := New(svc).HandleJob(context.Background(), queue.NewMonitorCheckJob(monitor.ID))
	if err == nil {
		t.Fatal("expected an internal error so the job can be retried")
	}
}

func TestHandleJobThreeFailuresOpenIncident(t *testing.T) {
	monitor := models.Monitor{
		ID:                 uuid.New(),
		Name:               "test",
		URL:                "https://example.com/down",
		HTTPMethod:         "GET",
		TimeoutSeconds:     2,
		ExpectedStatusCode: 200,
		IsActive:           true,
	}
	results := newMemoryCheckResults()
	incidents := newWorkerIncidents()
	checker := &seqFailChecker{}
	svc := service.NewMonitorCheckService(
		newMemoryMonitors(monitor),
		results,
		checker,
		nil,
		service.NewIncidentService(results, incidents, repository.NewMemoryTransactor()),
	)
	worker := New(svc)

	for i := 0; i < 3; i++ {
		if err := worker.HandleJob(context.Background(), queue.NewMonitorCheckJob(monitor.ID)); err != nil {
			t.Fatalf("HandleJob %d: %v", i, err)
		}
	}

	open, err := incidents.GetOpenByMonitorID(context.Background(), monitor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if open.FailureCount != 3 {
		t.Fatalf("failure_count = %d", open.FailureCount)
	}

	lastJob := queue.NewMonitorCheckJob(monitor.ID)
	if err := worker.HandleJob(context.Background(), lastJob); err != nil {
		t.Fatal(err)
	}
	if err := worker.HandleJob(context.Background(), lastJob); err != nil {
		t.Fatal(err)
	}

	listed, err := incidents.ListByMonitorID(context.Background(), monitor.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("len = %d, want 1 incident after duplicate delivery", len(listed))
	}
	if listed[0].FailureCount != 4 {
		t.Fatalf("failure_count = %d, want 4 after the fourth unique job plus duplicate", listed[0].FailureCount)
	}
}

type seqFailChecker struct {
	n int
}

func (s *seqFailChecker) Check(_ context.Context, monitor models.Monitor) models.CheckResult {
	s.n++
	status := 500
	return models.CheckResult{
		MonitorID:      monitor.ID,
		StatusCode:     &status,
		ResponseTimeMs: 20,
		Success:        false,
		CheckedAt:      time.Unix(int64(s.n), 0).UTC(),
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

func (m *memoryCheckResults) GetByJobID(_ context.Context, jobID uuid.UUID) (*models.CheckResult, error) {
	for _, result := range m.results {
		if result.JobID == jobID {
			copied := result
			return &copied, nil
		}
	}
	return nil, repository.ErrNotFound
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

type failingCheckResults struct {
	err error
}

func (f failingCheckResults) Create(_ context.Context, _ *models.CheckResult) error {
	return f.err
}

func (f failingCheckResults) GetByJobID(_ context.Context, _ uuid.UUID) (*models.CheckResult, error) {
	return nil, repository.ErrNotFound
}

func (f failingCheckResults) GetByID(_ context.Context, _ uuid.UUID) (*models.CheckResult, error) {
	return nil, repository.ErrNotFound
}

func (f failingCheckResults) ListByMonitorID(_ context.Context, _ uuid.UUID, _ int) ([]models.CheckResult, error) {
	return nil, nil
}

type workerIncidents struct {
	items map[uuid.UUID]models.Incident
}

func newWorkerIncidents() *workerIncidents {
	return &workerIncidents{items: make(map[uuid.UUID]models.Incident)}
}

func (m *workerIncidents) Create(_ context.Context, incident *models.Incident) error {
	if incident.ID == uuid.Nil {
		incident.ID = uuid.New()
	}
	now := time.Now().UTC()
	incident.CreatedAt = now
	incident.UpdatedAt = now
	if incident.Status == models.IncidentStatusOpen {
		for _, existing := range m.items {
			if existing.MonitorID == incident.MonitorID && existing.Status == models.IncidentStatusOpen {
				return repository.ErrDuplicate
			}
		}
	}
	m.items[incident.ID] = *incident
	return nil
}

func (m *workerIncidents) GetByID(_ context.Context, id uuid.UUID) (*models.Incident, error) {
	incident, ok := m.items[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copied := incident
	return &copied, nil
}

func (m *workerIncidents) GetOpenByMonitorID(_ context.Context, monitorID uuid.UUID) (*models.Incident, error) {
	for _, incident := range m.items {
		if incident.MonitorID == monitorID && incident.Status == models.IncidentStatusOpen {
			copied := incident
			return &copied, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *workerIncidents) ListByMonitorID(_ context.Context, monitorID uuid.UUID, limit int) ([]models.Incident, error) {
	listed := make([]models.Incident, 0)
	for _, incident := range m.items {
		if incident.MonitorID == monitorID {
			listed = append(listed, incident)
		}
	}
	if limit <= 0 || limit > len(listed) {
		return listed, nil
	}
	return listed[:limit], nil
}

func (m *workerIncidents) IncrementFailureCount(_ context.Context, id uuid.UUID, failureCount int) (*models.Incident, error) {
	incident, ok := m.items[id]
	if !ok || incident.Status != models.IncidentStatusOpen {
		return nil, repository.ErrNotFound
	}
	incident.FailureCount = failureCount
	m.items[id] = incident
	copied := incident
	return &copied, nil
}

func (m *workerIncidents) Resolve(_ context.Context, id uuid.UUID, resolvedAt time.Time) (*models.Incident, error) {
	incident, ok := m.items[id]
	if !ok || incident.Status != models.IncidentStatusOpen {
		return nil, repository.ErrNotFound
	}
	incident.Status = models.IncidentStatusResolved
	incident.ResolvedAt = &resolvedAt
	m.items[id] = incident
	copied := incident
	return &copied, nil
}
