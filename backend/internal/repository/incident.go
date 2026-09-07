package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

const (
	defaultIncidentLimit = 50
	maxIncidentLimit     = 100
)

// IncidentRepository stores outages detected from failed health checks.
type IncidentRepository interface {
	Create(ctx context.Context, incident *models.Incident) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Incident, error)
	GetOpenByMonitorID(ctx context.Context, monitorID uuid.UUID) (*models.Incident, error)
	ListByMonitorID(ctx context.Context, monitorID uuid.UUID, limit int) ([]models.Incident, error)
	IncrementFailureCount(ctx context.Context, id uuid.UUID, failureCount int) (*models.Incident, error)
	Resolve(ctx context.Context, id uuid.UUID, resolvedAt time.Time) (*models.Incident, error)
}

func clampIncidentLimit(limit int) int {
	if limit <= 0 {
		return defaultIncidentLimit
	}
	if limit > maxIncidentLimit {
		return maxIncidentLimit
	}
	return limit
}
