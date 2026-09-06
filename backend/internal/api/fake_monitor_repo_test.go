package api

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

// fakeMonitorRepo is an in-memory MonitorRepository used by HTTP tests.
// It lets us test handlers without PostgreSQL.
type fakeMonitorRepo struct {
	monitors map[uuid.UUID]models.Monitor
}

func newFakeMonitorRepo() *fakeMonitorRepo {
	return &fakeMonitorRepo{
		monitors: make(map[uuid.UUID]models.Monitor),
	}
}

func (f *fakeMonitorRepo) Create(_ context.Context, monitor *models.Monitor) error {
	if monitor.ID == uuid.Nil {
		monitor.ID = uuid.New()
	}
	now := time.Now().UTC()
	monitor.CreatedAt = now
	monitor.UpdatedAt = now
	f.monitors[monitor.ID] = *monitor
	return nil
}

func (f *fakeMonitorRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Monitor, error) {
	monitor, ok := f.monitors[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copied := monitor
	return &copied, nil
}

func (f *fakeMonitorRepo) List(_ context.Context) ([]models.Monitor, error) {
	monitors := make([]models.Monitor, 0, len(f.monitors))
	for _, monitor := range f.monitors {
		monitors = append(monitors, monitor)
	}
	return monitors, nil
}

func (f *fakeMonitorRepo) Update(_ context.Context, monitor *models.Monitor) error {
	existing, ok := f.monitors[monitor.ID]
	if !ok {
		return repository.ErrNotFound
	}
	monitor.CreatedAt = existing.CreatedAt
	monitor.UpdatedAt = time.Now().UTC()
	f.monitors[monitor.ID] = *monitor
	return nil
}

func (f *fakeMonitorRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.monitors[id]; !ok {
		return repository.ErrNotFound
	}
	delete(f.monitors, id)
	return nil
}
