package service

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
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

func TestPostgresIncidentServiceConcurrentWorkersOpenOnce(t *testing.T) {
	ctx := context.Background()
	pool, err := database.Connect(ctx, config.Load().DatabaseURL)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	monitors := repository.NewPostgresMonitorRepository(pool)
	checks := repository.NewPostgresCheckResultRepository(pool)
	incidents := repository.NewPostgresIncidentRepository(pool)
	svc := NewIncidentService(checks, incidents, repository.NewPostgresTransactor(pool))

	monitor := &models.Monitor{
		Name:                 "incident-race-" + uuid.NewString(),
		URL:                  "https://example.com/health",
		HTTPMethod:           "GET",
		CheckIntervalSeconds: 60,
		TimeoutSeconds:       5,
		ExpectedStatusCode:   200,
		IsActive:             true,
	}
	if err := monitors.Create(ctx, monitor); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = monitors.Delete(ctx, monitor.ID)
	})

	base := time.Now().UTC().Add(-time.Hour)
	for i := 0; i < 2; i++ {
		if err := checks.Create(ctx, failedResult(monitor.ID, base.Add(time.Duration(i)*time.Minute))); err != nil {
			t.Fatal(err)
		}
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			<-start
			result := failedResult(monitor.ID, base.Add(time.Duration(n+2)*time.Minute))
			if err := checks.Create(ctx, result); err != nil {
				errs <- err
				return
			}
			if err := svc.Evaluate(ctx, *result); err != nil {
				errs <- err
			}
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

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

func failedResult(monitorID uuid.UUID, at time.Time) *models.CheckResult {
	status := 500
	return &models.CheckResult{
		JobID:          uuid.New(),
		MonitorID:      monitorID,
		StatusCode:     &status,
		ResponseTimeMs: 25,
		Success:        false,
		CheckedAt:      at,
	}
}

func TestPostgresIncidentServiceDuplicateJobDoesNotDoubleCount(t *testing.T) {
	ctx := context.Background()
	pool, err := database.Connect(ctx, config.Load().DatabaseURL)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	monitors := repository.NewPostgresMonitorRepository(pool)
	checks := repository.NewPostgresCheckResultRepository(pool)
	incidents := repository.NewPostgresIncidentRepository(pool)
	svc := NewMonitorCheckService(
		monitors,
		checks,
		failingHTTPChecker{},
		nil,
		NewIncidentService(checks, incidents, repository.NewPostgresTransactor(pool)),
	)

	monitor := &models.Monitor{
		Name:                 "incident-dup-" + uuid.NewString(),
		URL:                  "https://example.com/health",
		HTTPMethod:           "GET",
		CheckIntervalSeconds: 60,
		TimeoutSeconds:       5,
		ExpectedStatusCode:   200,
		IsActive:             true,
	}
	if err := monitors.Create(ctx, monitor); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = monitors.Delete(ctx, monitor.ID)
	})

	var lastJob uuid.UUID
	for i := 0; i < 3; i++ {
		lastJob = uuid.New()
		if _, err := svc.RunCheck(ctx, monitor.ID, lastJob); err != nil {
			t.Fatal(err)
		}
	}

	open, err := incidents.GetOpenByMonitorID(ctx, monitor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if open.FailureCount != 3 {
		t.Fatalf("failure_count = %d, want 3", open.FailureCount)
	}

	if _, err := svc.RunCheck(ctx, monitor.ID, lastJob); err != nil {
		t.Fatal(err)
	}

	listed, err := incidents.ListByMonitorID(ctx, monitor.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("len = %d, want 1 incident", len(listed))
	}
	if listed[0].FailureCount != 3 {
		t.Fatalf("failure_count = %d after duplicate job, want 3", listed[0].FailureCount)
	}

	results, err := checks.ListByMonitorID(ctx, monitor.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("check results = %d, want 3", len(results))
	}
}

type failingHTTPChecker struct{}

func (failingHTTPChecker) Check(_ context.Context, monitor models.Monitor) models.CheckResult {
	status := 500
	return models.CheckResult{
		MonitorID:      monitor.ID,
		StatusCode:     &status,
		ResponseTimeMs: 20,
		Success:        false,
		CheckedAt:      time.Now().UTC(),
	}
}

func TestIncidentEvaluateFailureIsRetryableAfterCheckSaved(t *testing.T) {
	errBoom := errors.New("postgres unavailable")
	monitor := storedMonitor()
	checks := newMemoryCheckResults()
	svc := NewMonitorCheckService(
		newMemoryMonitors(monitor),
		checks,
		stubChecker{result: models.CheckResult{Success: false, CheckedAt: time.Now().UTC()}},
		nil,
		NewIncidentService(checks, failingIncidentRepo{err: errBoom}, repository.NewMemoryTransactor()),
	)

	_, err := svc.RunCheck(context.Background(), monitor.ID, uuid.New())
	if !errors.Is(err, errBoom) {
		t.Fatalf("err = %v, want %v so the job retries", err, errBoom)
	}
	listed, err := checks.ListByMonitorID(context.Background(), monitor.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("check was not saved before incident failure: %+v", listed)
	}
}

type failingIncidentRepo struct {
	err error
}

func (f failingIncidentRepo) Create(_ context.Context, _ *models.Incident) error { return f.err }
func (f failingIncidentRepo) GetByID(_ context.Context, _ uuid.UUID) (*models.Incident, error) {
	return nil, f.err
}
func (f failingIncidentRepo) GetOpenByMonitorID(_ context.Context, _ uuid.UUID) (*models.Incident, error) {
	return nil, f.err
}
func (f failingIncidentRepo) ListByMonitorID(_ context.Context, _ uuid.UUID, _ int) ([]models.Incident, error) {
	return nil, f.err
}
func (f failingIncidentRepo) IncrementFailureCount(_ context.Context, _ uuid.UUID, _ int) (*models.Incident, error) {
	return nil, f.err
}
func (f failingIncidentRepo) Resolve(_ context.Context, _ uuid.UUID, _ time.Time) (*models.Incident, error) {
	return nil, f.err
}
func (f failingIncidentRepo) CountByMonitorID(_ context.Context, _ uuid.UUID, _ time.Time) (int, error) {
	return 0, f.err
}
