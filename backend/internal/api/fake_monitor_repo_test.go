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
	if monitor.NextCheckAt == nil {
		monitor.NextCheckAt = &now
	}
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

func (f *fakeMonitorRepo) ListDue(_ context.Context, now time.Time, limit int) ([]models.Monitor, error) {
	due := make([]models.Monitor, 0)
	for _, monitor := range f.monitors {
		if monitor.IsActive && monitor.NextCheckAt != nil && !monitor.NextCheckAt.After(now) {
			due = append(due, monitor)
		}
	}
	if limit <= 0 || limit > len(due) {
		return due, nil
	}
	return due[:limit], nil
}

func (f *fakeMonitorRepo) ClaimDue(ctx context.Context, now time.Time, enqueue func(models.Monitor) error) (*models.Monitor, error) {
	due, err := f.ListDue(ctx, now, 1)
	if err != nil || len(due) == 0 {
		return nil, err
	}
	monitor := due[0]
	if err := enqueue(monitor); err != nil {
		return &monitor, err
	}
	next := repository.NextCheckTime(now, monitor.CheckIntervalSeconds)
	stored := f.monitors[monitor.ID]
	stored.NextCheckAt = &next
	f.monitors[monitor.ID] = stored
	monitor.NextCheckAt = &next
	return &monitor, nil
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
