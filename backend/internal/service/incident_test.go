package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

func TestOneFailureDoesNotOpenIncident(t *testing.T) {
	h := newIncidentHarness()
	h.save(t, false)

	if err := h.svc.Evaluate(context.Background(), h.latest(t)); err != nil {
		t.Fatal(err)
	}
	assertNoOpenIncident(t, h.incidents, h.monitor.ID)
}

func TestTwoFailuresDoNotOpenIncident(t *testing.T) {
	h := newIncidentHarness()
	h.save(t, false)
	h.save(t, false)

	if err := h.svc.Evaluate(context.Background(), h.latest(t)); err != nil {
		t.Fatal(err)
	}
	assertNoOpenIncident(t, h.incidents, h.monitor.ID)
}

func TestThirdConsecutiveFailureOpensIncident(t *testing.T) {
	h := newIncidentHarness()
	h.save(t, false)
	h.save(t, false)
	h.save(t, false)

	if err := h.svc.Evaluate(context.Background(), h.latest(t)); err != nil {
		t.Fatal(err)
	}

	open := assertOpenIncident(t, h.incidents, h.monitor.ID)
	if open.FailureCount != 3 {
		t.Fatalf("failure_count = %d, want 3", open.FailureCount)
	}
}

func TestFourthFailureDoesNotCreateSecondIncident(t *testing.T) {
	h := newIncidentHarness()
	h.save(t, false)
	h.save(t, false)
	h.save(t, false)
	if err := h.svc.Evaluate(context.Background(), h.latest(t)); err != nil {
		t.Fatal(err)
	}
	first := assertOpenIncident(t, h.incidents, h.monitor.ID)

	h.save(t, false)
	if err := h.svc.Evaluate(context.Background(), h.latest(t)); err != nil {
		t.Fatal(err)
	}

	listed, err := h.incidents.ListByMonitorID(context.Background(), h.monitor.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("len = %d, want 1 incident", len(listed))
	}
	if listed[0].ID != first.ID {
		t.Fatal("fourth failure opened a second incident")
	}
	if listed[0].FailureCount != 4 {
		t.Fatalf("failure_count = %d, want 4", listed[0].FailureCount)
	}
}

func TestOneSuccessDoesNotResolveIncident(t *testing.T) {
	h := openIncidentFromThreeFailures(t)

	h.save(t, true)
	if err := h.svc.Evaluate(context.Background(), h.latest(t)); err != nil {
		t.Fatal(err)
	}

	open := assertOpenIncident(t, h.incidents, h.monitor.ID)
	if open.FailureCount != 3 {
		t.Fatalf("failure_count = %d, want 3 after a single success", open.FailureCount)
	}
}

func TestSecondConsecutiveSuccessResolvesIncident(t *testing.T) {
	h := openIncidentFromThreeFailures(t)
	h.save(t, true)
	if err := h.svc.Evaluate(context.Background(), h.latest(t)); err != nil {
		t.Fatal(err)
	}
	h.save(t, true)
	if err := h.svc.Evaluate(context.Background(), h.latest(t)); err != nil {
		t.Fatal(err)
	}

	assertNoOpenIncident(t, h.incidents, h.monitor.ID)
	listed, err := h.incidents.ListByMonitorID(context.Background(), h.monitor.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Status != models.IncidentStatusResolved {
		t.Fatalf("listed = %+v", listed)
	}
	if listed[0].ResolvedAt == nil {
		t.Fatal("expected resolved_at")
	}
}

func TestNewFailuresAfterResolveOpenNewIncident(t *testing.T) {
	h := openIncidentFromThreeFailures(t)
	h.save(t, true)
	h.save(t, true)
	if err := h.svc.Evaluate(context.Background(), h.latest(t)); err != nil {
		t.Fatal(err)
	}
	first := mustListOne(t, h.incidents, h.monitor.ID)
	if first.Status != models.IncidentStatusResolved {
		t.Fatalf("status = %s", first.Status)
	}

	h.save(t, false)
	h.save(t, false)
	h.save(t, false)
	if err := h.svc.Evaluate(context.Background(), h.latest(t)); err != nil {
		t.Fatal(err)
	}

	listed, err := h.incidents.ListByMonitorID(context.Background(), h.monitor.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 {
		t.Fatalf("len = %d, want 2 incidents", len(listed))
	}
	open := assertOpenIncident(t, h.incidents, h.monitor.ID)
	if open.ID == first.ID {
		t.Fatal("resolved incident was reused")
	}
}

func TestDuplicateEvaluationDoesNotChangeIncident(t *testing.T) {
	h := openIncidentFromThreeFailures(t)
	before := assertOpenIncident(t, h.incidents, h.monitor.ID)

	if err := h.svc.Evaluate(context.Background(), h.latest(t)); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.Evaluate(context.Background(), h.latest(t)); err != nil {
		t.Fatal(err)
	}

	listed, err := h.incidents.ListByMonitorID(context.Background(), h.monitor.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("len = %d, want 1", len(listed))
	}
	if listed[0].ID != before.ID || listed[0].FailureCount != before.FailureCount {
		t.Fatalf("incident changed after duplicate evaluation: %+v", listed[0])
	}
}

type incidentHarness struct {
	svc       *IncidentService
	checks    *memoryCheckResults
	incidents *memoryIncidents
	monitor   models.Monitor
	seq       int
	base      time.Time
}

func newIncidentHarness() incidentHarness {
	monitor := storedMonitor()
	checks := newMemoryCheckResults()
	incidents := newMemoryIncidents()
	return incidentHarness{
		svc:       NewIncidentService(checks, incidents, repository.NewMemoryTransactor()),
		checks:    checks,
		incidents: incidents,
		monitor:   monitor,
		base:      time.Now().UTC().Add(-time.Hour),
	}
}

func openIncidentFromThreeFailures(t *testing.T) incidentHarness {
	t.Helper()
	h := newIncidentHarness()
	h.save(t, false)
	h.save(t, false)
	h.save(t, false)
	if err := h.svc.Evaluate(context.Background(), h.latest(t)); err != nil {
		t.Fatal(err)
	}
	assertOpenIncident(t, h.incidents, h.monitor.ID)
	return h
}

func (h *incidentHarness) save(t *testing.T, success bool) {
	t.Helper()
	h.seq++
	result := models.CheckResult{
		JobID:     uuid.New(),
		MonitorID: h.monitor.ID,
		Success:   success,
		CheckedAt: h.base.Add(time.Duration(h.seq) * time.Minute),
	}
	if !success {
		status := 500
		result.StatusCode = &status
	}
	if err := h.checks.Create(context.Background(), &result); err != nil {
		t.Fatal(err)
	}
}

func (h *incidentHarness) latest(t *testing.T) models.CheckResult {
	t.Helper()
	listed, err := h.checks.ListByMonitorID(context.Background(), h.monitor.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) == 0 {
		t.Fatal("no checks")
	}
	return listed[0]
}

func assertOpenIncident(t *testing.T, repo *memoryIncidents, monitorID uuid.UUID) *models.Incident {
	t.Helper()
	open, err := repo.GetOpenByMonitorID(context.Background(), monitorID)
	if err != nil {
		t.Fatalf("open incident: %v", err)
	}
	if open.Status != models.IncidentStatusOpen {
		t.Fatalf("status = %s", open.Status)
	}
	return open
}

func assertNoOpenIncident(t *testing.T, repo *memoryIncidents, monitorID uuid.UUID) {
	t.Helper()
	_, err := repo.GetOpenByMonitorID(context.Background(), monitorID)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func mustListOne(t *testing.T, repo *memoryIncidents, monitorID uuid.UUID) models.Incident {
	t.Helper()
	listed, err := repo.ListByMonitorID(context.Background(), monitorID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("len = %d, want 1", len(listed))
	}
	return listed[0]
}

type memoryIncidents struct {
	mu    sync.Mutex
	items map[uuid.UUID]models.Incident
}

func newMemoryIncidents() *memoryIncidents {
	return &memoryIncidents{items: make(map[uuid.UUID]models.Incident)}
}

func (m *memoryIncidents) Create(_ context.Context, incident *models.Incident) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if incident.ID == uuid.Nil {
		incident.ID = uuid.New()
	}
	now := time.Now().UTC()
	if incident.StartedAt.IsZero() {
		incident.StartedAt = now
	}
	if incident.Status == "" {
		incident.Status = models.IncidentStatusOpen
	}
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

func (m *memoryIncidents) GetByID(_ context.Context, id uuid.UUID) (*models.Incident, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	incident, ok := m.items[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copied := incident
	return &copied, nil
}

func (m *memoryIncidents) GetOpenByMonitorID(_ context.Context, monitorID uuid.UUID) (*models.Incident, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, incident := range m.items {
		if incident.MonitorID == monitorID && incident.Status == models.IncidentStatusOpen {
			copied := incident
			return &copied, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *memoryIncidents) ListByMonitorID(_ context.Context, monitorID uuid.UUID, limit int) ([]models.Incident, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	listed := make([]models.Incident, 0)
	for _, incident := range m.items {
		if incident.MonitorID == monitorID {
			listed = append(listed, incident)
		}
	}
	for i := 0; i < len(listed); i++ {
		for j := i + 1; j < len(listed); j++ {
			if listed[j].StartedAt.After(listed[i].StartedAt) {
				listed[i], listed[j] = listed[j], listed[i]
			}
		}
	}
	if limit <= 0 || limit > len(listed) {
		return listed, nil
	}
	return listed[:limit], nil
}

func (m *memoryIncidents) IncrementFailureCount(_ context.Context, id uuid.UUID, failureCount int) (*models.Incident, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	incident, ok := m.items[id]
	if !ok || incident.Status != models.IncidentStatusOpen {
		return nil, repository.ErrNotFound
	}
	incident.FailureCount = failureCount
	incident.UpdatedAt = time.Now().UTC()
	m.items[id] = incident
	copied := incident
	return &copied, nil
}

func (m *memoryIncidents) Resolve(_ context.Context, id uuid.UUID, resolvedAt time.Time) (*models.Incident, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	incident, ok := m.items[id]
	if !ok || incident.Status != models.IncidentStatusOpen {
		return nil, repository.ErrNotFound
	}
	incident.Status = models.IncidentStatusResolved
	incident.ResolvedAt = &resolvedAt
	incident.UpdatedAt = time.Now().UTC()
	m.items[id] = incident
	copied := incident
	return &copied, nil
}
