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

func TestPostgresIncidentRepository(t *testing.T) {
	ctx := context.Background()
	pool, err := database.Connect(ctx, config.Load().DatabaseURL)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	monitors := NewPostgresMonitorRepository(pool)
	incidents := NewPostgresIncidentRepository(pool)
	monitor := createRepoTestMonitor(t, ctx, monitors)

	t.Run("create get and list newest first", func(t *testing.T) {
		older := createRepoTestIncident(t, ctx, incidents, monitor.ID, models.IncidentStatusResolved, time.Now().UTC().Add(-time.Hour))
		newer := createRepoTestIncident(t, ctx, incidents, monitor.ID, models.IncidentStatusOpen, time.Now().UTC())

		got, err := incidents.GetByID(ctx, newer.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.MonitorID != monitor.ID || got.Status != models.IncidentStatusOpen {
			t.Fatalf("got = %+v", got)
		}

		listed, err := incidents.ListByMonitorID(ctx, monitor.ID, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(listed) < 2 || listed[0].ID != newer.ID || listed[1].ID != older.ID {
			t.Fatalf("listed = %+v", listed)
		}

		open, err := incidents.GetOpenByMonitorID(ctx, monitor.ID)
		if err != nil {
			t.Fatal(err)
		}
		if open.ID != newer.ID {
			t.Fatalf("open = %s, want %s", open.ID, newer.ID)
		}
	})

	t.Run("one open incident per monitor", func(t *testing.T) {
		only := createRepoTestMonitor(t, ctx, monitors)
		_ = createRepoTestIncident(t, ctx, incidents, only.ID, models.IncidentStatusOpen, time.Now().UTC())

		duplicate := &models.Incident{
			MonitorID:    only.ID,
			StartedAt:    time.Now().UTC(),
			Status:       models.IncidentStatusOpen,
			FailureCount: 3,
		}
		err := incidents.Create(ctx, duplicate)
		if !errors.Is(err, ErrDuplicate) {
			t.Fatalf("err = %v, want ErrDuplicate", err)
		}
	})

	t.Run("increment failure count and resolve", func(t *testing.T) {
		only := createRepoTestMonitor(t, ctx, monitors)
		open := createRepoTestIncident(t, ctx, incidents, only.ID, models.IncidentStatusOpen, time.Now().UTC())

		updated, err := incidents.IncrementFailureCount(ctx, open.ID, 4)
		if err != nil {
			t.Fatal(err)
		}
		if updated.FailureCount != 4 {
			t.Fatalf("failure_count = %d", updated.FailureCount)
		}

		resolvedAt := time.Now().UTC()
		resolved, err := incidents.Resolve(ctx, open.ID, resolvedAt)
		if err != nil {
			t.Fatal(err)
		}
		if resolved.Status != models.IncidentStatusResolved || resolved.ResolvedAt == nil {
			t.Fatalf("resolved = %+v", resolved)
		}

		_, err = incidents.GetOpenByMonitorID(ctx, only.ID)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("delete monitor cascades incidents", func(t *testing.T) {
		only := createRepoTestMonitor(t, ctx, monitors)
		saved := createRepoTestIncident(t, ctx, incidents, only.ID, models.IncidentStatusOpen, time.Now().UTC())

		if err := monitors.Delete(ctx, only.ID); err != nil {
			t.Fatal(err)
		}
		_, err := incidents.GetByID(ctx, saved.ID)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound after cascade", err)
		}
	})
}

func TestPostgresConcurrentEvaluateCannotOpenTwoIncidents(t *testing.T) {
	ctx := context.Background()
	pool, err := database.Connect(ctx, config.Load().DatabaseURL)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	monitors := NewPostgresMonitorRepository(pool)
	results := NewPostgresCheckResultRepository(pool)
	incidents := NewPostgresIncidentRepository(pool)
	monitor := createRepoTestMonitor(t, ctx, monitors)

	base := time.Now().UTC().Add(-time.Hour)
	for i := 0; i < 2; i++ {
		status := 500
		result := &models.CheckResult{
			JobID:          uuid.New(),
			MonitorID:      monitor.ID,
			StatusCode:     &status,
			ResponseTimeMs: 10,
			Success:        false,
			CheckedAt:      base.Add(time.Duration(i) * time.Minute),
		}
		if err := results.Create(ctx, result); err != nil {
			t.Fatal(err)
		}
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			<-start
			status := 500
			result := &models.CheckResult{
				JobID:          uuid.New(),
				MonitorID:      monitor.ID,
				StatusCode:     &status,
				ResponseTimeMs: 10,
				Success:        false,
				CheckedAt:      base.Add(time.Duration(n+2) * time.Minute),
			}
			if err := results.Create(ctx, result); err != nil {
				t.Errorf("create check: %v", err)
				return
			}

			tx := NewPostgresTransactor(pool)
			err := tx.WithMonitorLock(ctx, monitor.ID, func(ctx context.Context) error {
				open, err := incidents.GetOpenByMonitorID(ctx, monitor.ID)
				if err != nil && !errors.Is(err, ErrNotFound) {
					return err
				}
				if open != nil {
					_, err := incidents.IncrementFailureCount(ctx, open.ID, open.FailureCount+1)
					return err
				}
				incident := &models.Incident{
					MonitorID:    monitor.ID,
					StartedAt:    result.CheckedAt,
					Status:       models.IncidentStatusOpen,
					FailureCount: 3,
				}
				if err := incidents.Create(ctx, incident); err != nil && !errors.Is(err, ErrDuplicate) {
					return err
				}
				return nil
			})
			if err != nil {
				t.Errorf("evaluate: %v", err)
			}
		}(i)
	}
	close(start)
	wg.Wait()

	listed, err := incidents.ListByMonitorID(ctx, monitor.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	openCount := 0
	for _, incident := range listed {
		if incident.Status == models.IncidentStatusOpen {
			openCount++
		}
	}
	if openCount != 1 {
		t.Fatalf("open incidents = %d, want 1; listed = %+v", openCount, listed)
	}
}

func createRepoTestIncident(t *testing.T, ctx context.Context, incidents *PostgresIncidentRepository, monitorID uuid.UUID, status string, startedAt time.Time) *models.Incident {
	t.Helper()
	incident := &models.Incident{
		MonitorID:    monitorID,
		StartedAt:    startedAt,
		Status:       status,
		FailureCount: 3,
	}
	if status == models.IncidentStatusResolved {
		resolved := startedAt.Add(time.Minute)
		incident.ResolvedAt = &resolved
	}
	if err := incidents.Create(ctx, incident); err != nil {
		t.Fatalf("create incident: %v", err)
	}
	return incident
}
