package api

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

type fakeIncidentRepo struct {
	mu        sync.Mutex
	incidents map[uuid.UUID]models.Incident
}

func newFakeIncidentRepo() *fakeIncidentRepo {
	return &fakeIncidentRepo{incidents: make(map[uuid.UUID]models.Incident)}
}

func (f *fakeIncidentRepo) Create(_ context.Context, incident *models.Incident) error {
	f.mu.Lock()
	defer f.mu.Unlock()
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
		for _, existing := range f.incidents {
			if existing.MonitorID == incident.MonitorID && existing.Status == models.IncidentStatusOpen {
				return repository.ErrDuplicate
			}
		}
	}
	f.incidents[incident.ID] = *incident
	return nil
}

func (f *fakeIncidentRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Incident, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	incident, ok := f.incidents[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copied := incident
	return &copied, nil
}

func (f *fakeIncidentRepo) GetOpenByMonitorID(_ context.Context, monitorID uuid.UUID) (*models.Incident, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, incident := range f.incidents {
		if incident.MonitorID == monitorID && incident.Status == models.IncidentStatusOpen {
			copied := incident
			return &copied, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeIncidentRepo) ListByMonitorID(_ context.Context, monitorID uuid.UUID, limit int) ([]models.Incident, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	listed := make([]models.Incident, 0)
	for _, incident := range f.incidents {
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
	if limit <= 0 {
		limit = 50
	}
	if limit > len(listed) {
		return listed, nil
	}
	return listed[:limit], nil
}

func (f *fakeIncidentRepo) IncrementFailureCount(_ context.Context, id uuid.UUID, failureCount int) (*models.Incident, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	incident, ok := f.incidents[id]
	if !ok || incident.Status != models.IncidentStatusOpen {
		return nil, repository.ErrNotFound
	}
	incident.FailureCount = failureCount
	incident.UpdatedAt = time.Now().UTC()
	f.incidents[id] = incident
	copied := incident
	return &copied, nil
}

func (f *fakeIncidentRepo) Resolve(_ context.Context, id uuid.UUID, resolvedAt time.Time) (*models.Incident, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	incident, ok := f.incidents[id]
	if !ok || incident.Status != models.IncidentStatusOpen {
		return nil, repository.ErrNotFound
	}
	incident.Status = models.IncidentStatusResolved
	incident.ResolvedAt = &resolvedAt
	incident.UpdatedAt = time.Now().UTC()
	f.incidents[id] = incident
	copied := incident
	return &copied, nil
}
