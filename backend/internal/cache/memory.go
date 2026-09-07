package cache

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

// Memory is an in-process stats cache for tests.
type Memory struct {
	mu      sync.Mutex
	items   map[uuid.UUID]models.MonitorStats
	GetErr  error
	SetErr  error
	DelErr  error
	Gets    []uuid.UUID
	Sets    []uuid.UUID
	Deletes []uuid.UUID
}

func NewMemory() *Memory {
	return &Memory{items: make(map[uuid.UUID]models.MonitorStats)}
}

func (m *Memory) GetMonitorStats(_ context.Context, monitorID uuid.UUID) (*models.MonitorStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Gets = append(m.Gets, monitorID)
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	stats, ok := m.items[monitorID]
	if !ok {
		return nil, nil
	}
	copied := stats
	return &copied, nil
}

func (m *Memory) SetMonitorStats(_ context.Context, monitorID uuid.UUID, stats models.MonitorStats, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Sets = append(m.Sets, monitorID)
	if m.SetErr != nil {
		return m.SetErr
	}
	m.items[monitorID] = stats
	return nil
}

func (m *Memory) DeleteMonitorStats(_ context.Context, monitorID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Deletes = append(m.Deletes, monitorID)
	if m.DelErr != nil {
		return m.DelErr
	}
	delete(m.items, monitorID)
	return nil
}
