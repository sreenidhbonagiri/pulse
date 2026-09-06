package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

var ErrNotFound = errors.New("not found")

// MonitorRepository is the database operations Pulse needs for monitors.
type MonitorRepository interface {
	Create(ctx context.Context, monitor *models.Monitor) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Monitor, error)
	List(ctx context.Context) ([]models.Monitor, error)
	Update(ctx context.Context, monitor *models.Monitor) error
	Delete(ctx context.Context, id uuid.UUID) error
}
