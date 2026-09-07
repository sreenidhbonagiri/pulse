package cache

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

const DefaultTTL = 45 * time.Second

// MonitorStatsCache stores precomputed dashboard stats.
// Handlers should depend on this interface, not on Redis.
type MonitorStatsCache interface {
	GetMonitorStats(ctx context.Context, monitorID uuid.UUID) (*models.MonitorStats, error)
	SetMonitorStats(ctx context.Context, monitorID uuid.UUID, stats models.MonitorStats, ttl time.Duration) error
	DeleteMonitorStats(ctx context.Context, monitorID uuid.UUID) error
}

func statsKey(monitorID uuid.UUID) string {
	return "pulse:monitor:" + monitorID.String() + ":stats"
}

// InvalidateMonitorStats removes cached stats. Redis errors are logged and ignored
// so a cache outage cannot fail a check or API request.
func InvalidateMonitorStats(ctx context.Context, c MonitorStatsCache, monitorID uuid.UUID) {
	if c == nil {
		return
	}
	if err := c.DeleteMonitorStats(ctx, monitorID); err != nil {
		log.Printf("stats cache delete monitor_id=%s: %v", monitorID, err)
	}
}
