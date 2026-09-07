package scheduler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

func TestTickSchedulesDueActiveMonitor(t *testing.T) {
	now := time.Now().UTC()
	due := newTestMonitor(true, ptrTime(now.Add(-time.Second)))
	repo := newMemoryMonitors(due)
	publisher := queue.NewMemoryPublisher()

	New(repo, publisher).Tick(context.Background(), now)

	if len(publisher.Jobs) != 1 || publisher.Jobs[0].MonitorID != due.ID {
		t.Fatalf("jobs = %+v", publisher.Jobs)
	}
	got, _ := repo.GetByID(context.Background(), due.ID)
	if got.NextCheckAt == nil || !got.NextCheckAt.After(now) {
		t.Fatalf("next_check_at = %v, want in the future", got.NextCheckAt)
	}
}

func TestTickSkipsMonitorNotYetDue(t *testing.T) {
	now := time.Now().UTC()
	future := newTestMonitor(true, ptrTime(now.Add(time.Hour)))
	repo := newMemoryMonitors(future)
	publisher := queue.NewMemoryPublisher()

	New(repo, publisher).Tick(context.Background(), now)

	if len(publisher.Jobs) != 0 {
		t.Fatalf("published %d jobs, want 0", len(publisher.Jobs))
	}
}

func TestTickSkipsInactiveMonitor(t *testing.T) {
	now := time.Now().UTC()
	inactive := newTestMonitor(false, ptrTime(now.Add(-time.Second)))
	repo := newMemoryMonitors(inactive)
	publisher := queue.NewMemoryPublisher()

	New(repo, publisher).Tick(context.Background(), now)

	if len(publisher.Jobs) != 0 {
		t.Fatalf("published %d jobs, want 0", len(publisher.Jobs))
	}
}

func TestTickPublishFailureDoesNotAdvanceNextCheckAt(t *testing.T) {
	now := time.Now().UTC()
	original := now.Add(-time.Second)
	due := newTestMonitor(true, ptrTime(original))
	repo := newMemoryMonitors(due)
	publisher := queue.NewMemoryPublisher()
	publisher.Err = errors.New("rabbitmq down")

	New(repo, publisher).Tick(context.Background(), now)

	got, _ := repo.GetByID(context.Background(), due.ID)
	if got.NextCheckAt == nil || !got.NextCheckAt.Equal(original) {
		t.Fatalf("next_check_at = %v, want original %v", got.NextCheckAt, original)
	}
}

func TestTickPreventsDuplicateClaim(t *testing.T) {
	now := time.Now().UTC()
	due := newTestMonitor(true, ptrTime(now.Add(-time.Second)))
	repo := newMemoryMonitors(due)

	started := make(chan struct{})
	release := make(chan struct{})
	first := queue.NewMemoryPublisher()
	second := queue.NewMemoryPublisher()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		New(repo, first).monitors.ClaimDue(context.Background(), now, func(models.Monitor) error {
			close(started)
			<-release
			return first.Publish(context.Background(), queue.NewMonitorCheckJob(due.ID))
		})
	}()

	<-started
	monitor, err := New(repo, second).monitors.ClaimDue(context.Background(), now, func(models.Monitor) error {
		t.Error("second scheduler should not enqueue the locked monitor")
		return second.Publish(context.Background(), queue.NewMonitorCheckJob(due.ID))
	})
	if err != nil {
		t.Fatalf("second ClaimDue: %v", err)
	}
	if monitor != nil {
		t.Fatal("second scheduler claimed the same due monitor")
	}

	close(release)
	wg.Wait()
	if len(second.Jobs) != 0 {
		t.Fatalf("second publisher jobs = %+v", second.Jobs)
	}
}

func newTestMonitor(active bool, next *time.Time) models.Monitor {
	return models.Monitor{
		ID:                   uuid.New(),
		Name:                 "sched-test",
		URL:                  "https://example.com/health",
		HTTPMethod:           "GET",
		CheckIntervalSeconds: 60,
		TimeoutSeconds:       5,
		ExpectedStatusCode:   200,
		IsActive:             active,
		NextCheckAt:          next,
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

type memoryMonitors struct {
	mu       sync.Mutex
	monitors map[uuid.UUID]models.Monitor
	claiming map[uuid.UUID]bool
}

func newMemoryMonitors(items ...models.Monitor) *memoryMonitors {
	repo := &memoryMonitors{
		monitors: make(map[uuid.UUID]models.Monitor),
		claiming: make(map[uuid.UUID]bool),
	}
	for _, item := range items {
		repo.monitors[item.ID] = item
	}
	return repo
}

func (m *memoryMonitors) Create(_ context.Context, monitor *models.Monitor) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.monitors[monitor.ID] = *monitor
	return nil
}

func (m *memoryMonitors) GetByID(_ context.Context, id uuid.UUID) (*models.Monitor, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
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

func (m *memoryMonitors) ListDue(_ context.Context, now time.Time, limit int) ([]models.Monitor, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dueLocked(now, limit, nil), nil
}

func (m *memoryMonitors) ClaimDue(_ context.Context, now time.Time, enqueue func(models.Monitor) error) (*models.Monitor, error) {
	m.mu.Lock()
	due := m.dueLocked(now, 1, m.claiming)
	if len(due) == 0 {
		m.mu.Unlock()
		return nil, nil
	}
	monitor := due[0]
	m.claiming[monitor.ID] = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.claiming, monitor.ID)
		m.mu.Unlock()
	}()

	if err := enqueue(monitor); err != nil {
		return &monitor, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	next := repository.NextCheckTime(now, monitor.CheckIntervalSeconds)
	stored := m.monitors[monitor.ID]
	stored.NextCheckAt = &next
	m.monitors[monitor.ID] = stored
	monitor.NextCheckAt = &next
	return &monitor, nil
}

func (m *memoryMonitors) dueLocked(now time.Time, limit int, skip map[uuid.UUID]bool) []models.Monitor {
	due := make([]models.Monitor, 0)
	for _, monitor := range m.monitors {
		if skip[monitor.ID] {
			continue
		}
		if !monitor.IsActive || monitor.NextCheckAt == nil || monitor.NextCheckAt.After(now) {
			continue
		}
		due = append(due, monitor)
	}
	for i := 0; i < len(due); i++ {
		for j := i + 1; j < len(due); j++ {
			if due[j].NextCheckAt.Before(*due[i].NextCheckAt) {
				due[i], due[j] = due[j], due[i]
			}
		}
	}
	limit = repositoryClamp(limit)
	if limit > len(due) {
		return due
	}
	return due[:limit]
}

func repositoryClamp(limit int) int {
	if limit <= 0 {
		return 50
	}
	return limit
}

func (m *memoryMonitors) Update(_ context.Context, monitor *models.Monitor) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.monitors[monitor.ID] = *monitor
	return nil
}

func (m *memoryMonitors) Delete(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.monitors, id)
	return nil
}
