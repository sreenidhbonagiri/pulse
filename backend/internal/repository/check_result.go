package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

const (
	defaultCheckResultLimit = 50
	maxCheckResultLimit     = 100
)

// CheckResultRepository stores HTTP check outcomes.
type CheckResultRepository interface {
	Create(ctx context.Context, result *models.CheckResult) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.CheckResult, error)
	GetByJobID(ctx context.Context, jobID uuid.UUID) (*models.CheckResult, error)
	ListByMonitorID(ctx context.Context, monitorID uuid.UUID, limit int) ([]models.CheckResult, error)
	GetCheckStats(ctx context.Context, monitorID uuid.UUID, since time.Time) (CheckStats, error)
}

func clampCheckResultLimit(limit int) int {
	if limit <= 0 {
		return defaultCheckResultLimit
	}
	if limit > maxCheckResultLimit {
		return maxCheckResultLimit
	}
	return limit
}
