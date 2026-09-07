package cache

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

func TestRedisStatsCache(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	c, err := Dial(ctx, config.Load().RedisURL)
	if err != nil {
		t.Skipf("redis not available: %v", err)
	}
	defer c.Close()

	if err := c.client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available: %v", err)
	}

	id := uuid.New()
	t.Cleanup(func() {
		_ = c.DeleteMonitorStats(context.Background(), id)
	})

	got, err := c.GetMonitorStats(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expected miss")
	}

	want := models.MonitorStats{CurrentStatus: models.MonitorStatusDown, TotalChecks: 3, FailedChecks: 1}
	if err := c.SetMonitorStats(ctx, id, want, time.Minute); err != nil {
		t.Fatal(err)
	}

	got, err = c.GetMonitorStats(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.TotalChecks != 3 || got.CurrentStatus != models.MonitorStatusDown {
		t.Fatalf("got = %+v", got)
	}

	if err := c.DeleteMonitorStats(ctx, id); err != nil {
		t.Fatal(err)
	}
	got, err = c.GetMonitorStats(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expected miss after delete")
	}
}
