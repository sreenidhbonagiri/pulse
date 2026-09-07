package cache

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

func TestMemoryCacheHitMissSetDelete(t *testing.T) {
	c := NewMemory()
	id := uuid.New()
	ctx := context.Background()

	got, err := c.GetMonitorStats(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expected cache miss")
	}

	stats := models.MonitorStats{CurrentStatus: models.MonitorStatusUp, TotalChecks: 4}
	if err := c.SetMonitorStats(ctx, id, stats, DefaultTTL); err != nil {
		t.Fatal(err)
	}

	got, err = c.GetMonitorStats(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.TotalChecks != 4 || got.CurrentStatus != models.MonitorStatusUp {
		t.Fatalf("cache hit = %+v", got)
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

func TestInvalidateMonitorStatsLogsAndIgnoresErrors(t *testing.T) {
	c := NewMemory()
	c.DelErr = errors.New("redis down")
	id := uuid.New()
	_ = c.SetMonitorStats(context.Background(), id, models.MonitorStats{TotalChecks: 1}, DefaultTTL)

	InvalidateMonitorStats(context.Background(), c, id)
	if len(c.Deletes) != 1 {
		t.Fatalf("deletes = %d", len(c.Deletes))
	}
	got, err := c.GetMonitorStats(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("failed delete should leave the in-memory value when DelErr is set")
	}
}

func TestStatsKey(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	if statsKey(id) != "pulse:monitor:11111111-1111-1111-1111-111111111111:stats" {
		t.Fatalf("key = %s", statsKey(id))
	}
}
