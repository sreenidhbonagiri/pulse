package models

import (
	"time"

	"github.com/google/uuid"
)

// CheckResult is the outcome of one HTTP health check against a Monitor.
type CheckResult struct {
	ID             uuid.UUID `json:"id"`
	MonitorID      uuid.UUID `json:"monitor_id"`
	StatusCode     *int      `json:"status_code"`
	ResponseTimeMs int       `json:"response_time_ms"`
	Success        bool      `json:"success"`
	ErrorMessage   *string   `json:"error_message"`
	CheckedAt      time.Time `json:"checked_at"`
}
