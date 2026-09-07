package repository

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
	"github.com/sreenidhbonagiri/pulse/backend/internal/database"
	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

func TestPostgresMonitorListDueAndClaim(t *testing.T) {
	ctx := context.Background()
	pool, err := database.Connect(ctx, config.Load().DatabaseURL)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewPostgresMonitorRepository(pool)
	now := time.Now().UTC()
	dueAt := time.Unix(0, 0).UTC()
	due := createScheduledMonitor(t, ctx, repo, true, dueAt)
	future := createScheduledMonitor(t, ctx, repo, true, now.Add(time.Hour))
	inactive := createScheduledMonitor(t, ctx, repo, false, dueAt)

	t.Run("list due skips future and inactive", func(t *testing.T) {
		listed, err := repo.ListDue(ctx, now, 50)
		if err != nil {
			t.Fatalf("ListDue: %v", err)
		}
		if !containsMonitor(listed, due.ID) {
			t.Fatal("expected due monitor")
		}
		if containsMonitor(listed, future.ID) {
			t.Fatal("future monitor should be skipped")
		}
		if containsMonitor(listed, inactive.ID) {
			t.Fatal("inactive monitor should be skipped")
		}
	})

	t.Run("claim advances next_check_at after successful enqueue", func(t *testing.T) {
		enqueued := false
		got, err := repo.ClaimDue(ctx, now, func(models.Monitor) error {
			enqueued = true
			return nil
		})
		if err != nil {
			t.Fatalf("ClaimDue: %v", err)
		}
		if got == nil || got.ID != due.ID {
			t.Fatalf("claimed = %+v", got)
		}
		if !enqueued {
			t.Fatal("expected enqueue to run")
		}
		saved, err := repo.GetByID(ctx, due.ID)
		if err != nil {
			t.Fatal(err)
		}
		if saved.NextCheckAt == nil || !saved.NextCheckAt.After(now) {
			t.Fatalf("next_check_at = %v", saved.NextCheckAt)
		}
	})
}

func TestPostgresClaimDuePublishFailureDoesNotAdvance(t *testing.T) {
	ctx := context.Background()
	pool, err := database.Connect(ctx, config.Load().DatabaseURL)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewPostgresMonitorRepository(pool)
	now := time.Now().UTC()
	original := time.Unix(1, 0).UTC()
	due := createScheduledMonitor(t, ctx, repo, true, original)

	_, err = repo.ClaimDue(ctx, now, func(models.Monitor) error {
		return errors.New("publish failed")
	})
	if err == nil {
		t.Fatal("expected publish error")
	}

	saved, err := repo.GetByID(ctx, due.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.NextCheckAt == nil || saved.NextCheckAt.Sub(original) > time.Second {
		t.Fatalf("next_check_at changed to %v, want %v", saved.NextCheckAt, original)
	}
}

func TestPostgresClaimDueSkipLocked(t *testing.T) {
	ctx := context.Background()
	pool, err := database.Connect(ctx, config.Load().DatabaseURL)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewPostgresMonitorRepository(pool)
	now := time.Now().UTC()
	due := createScheduledMonitor(t, ctx, repo, true, time.Unix(2, 0).UTC())

	started := make(chan struct{})
	release := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := repo.ClaimDue(ctx, now, func(models.Monitor) error {
			close(started)
			<-release
			return nil
		})
		if err != nil {
			t.Errorf("first ClaimDue: %v", err)
		}
	}()

	<-started
	second, err := repo.ClaimDue(ctx, now, func(models.Monitor) error {
		return nil
	})
	if err != nil {
		t.Fatalf("second ClaimDue: %v", err)
	}
	if second != nil && second.ID == due.ID {
		t.Fatal("two schedulers claimed the same due monitor")
	}

	close(release)
	wg.Wait()
}

func createScheduledMonitor(t *testing.T, ctx context.Context, repo *PostgresMonitorRepository, active bool, next time.Time) *models.Monitor {
	t.Helper()
	monitor := &models.Monitor{
		Name:                 "due-test-" + uuid.NewString(),
		URL:                  "https://example.com/health",
		HTTPMethod:           "GET",
		CheckIntervalSeconds: 60,
		TimeoutSeconds:       5,
		ExpectedStatusCode:   200,
		IsActive:             active,
		NextCheckAt:          &next,
	}
	if err := repo.Create(ctx, monitor); err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Delete(ctx, monitor.ID)
	})
	return monitor
}

func containsMonitor(monitors []models.Monitor, id uuid.UUID) bool {
	for _, monitor := range monitors {
		if monitor.ID == id {
			return true
		}
	}
	return false
}
