package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	IncidentStatusOpen     = "open"
	IncidentStatusResolved = "resolved"
)

// Incident is an outage detected from consecutive failed health checks.
type Incident struct {
	ID           uuid.UUID  `json:"id"`
	MonitorID    uuid.UUID  `json:"monitor_id"`
	StartedAt    time.Time  `json:"started_at"`
	ResolvedAt   *time.Time `json:"resolved_at"`
	Status       string     `json:"status"`
	FailureCount int        `json:"failure_count"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
