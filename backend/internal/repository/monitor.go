package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

var ErrNotFound = errors.New("not found")
var ErrDuplicate = errors.New("duplicate")

const (
	defaultDueLimit = 50
	maxDueLimit     = 100
)

// MonitorRepository is the database operations Pulse needs for monitors.
type MonitorRepository interface {
	Create(ctx context.Context, monitor *models.Monitor) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Monitor, error)
	List(ctx context.Context) ([]models.Monitor, error)
	ListDue(ctx context.Context, now time.Time, limit int) ([]models.Monitor, error)
	ClaimDue(ctx context.Context, now time.Time, enqueue func(models.Monitor) error) (*models.Monitor, error)
	Update(ctx context.Context, monitor *models.Monitor) error
	Delete(ctx context.Context, id uuid.UUID) error
}

func clampDueLimit(limit int) int {
	if limit <= 0 {
		return defaultDueLimit
	}
	if limit > maxDueLimit {
		return maxDueLimit
	}
	return limit
}

func NextCheckTime(from time.Time, intervalSeconds int) time.Time {
	if intervalSeconds <= 0 {
		intervalSeconds = 60
	}
	return from.Add(time.Duration(intervalSeconds) * time.Second)
}
