package models

import (
	"time"

	"github.com/google/uuid"
)

// Monitor is a website or API endpoint that Pulse will check later.
type Monitor struct {
	ID                   uuid.UUID  `json:"id"`
	UserID               *uuid.UUID `json:"user_id"`
	Name                 string     `json:"name"`
	URL                  string     `json:"url"`
	HTTPMethod           string     `json:"http_method"`
	CheckIntervalSeconds int        `json:"check_interval_seconds"`
	TimeoutSeconds       int        `json:"timeout_seconds"`
	ExpectedStatusCode   int        `json:"expected_status_code"`
	IsActive             bool       `json:"is_active"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}
