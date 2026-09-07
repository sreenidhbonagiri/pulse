package repository

import (
	"context"

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
	ListByMonitorID(ctx context.Context, monitorID uuid.UUID, limit int) ([]models.CheckResult, error)
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
