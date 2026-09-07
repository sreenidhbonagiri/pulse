package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/sreenidhbonagiri/pulse/backend/internal/metrics"
	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
	"github.com/sreenidhbonagiri/pulse/backend/internal/service"
)

func TestHandleJobIncrementsProcessedAndCheckCounters(t *testing.T) {
	m := metrics.New("worker")

	monitor := models.Monitor{
		ID:                 uuid.New(),
		Name:               "test",
		URL:                "https://example.com/health",
		HTTPMethod:         "GET",
		TimeoutSeconds:     2,
		ExpectedStatusCode: 200,
		IsActive:           true,
	}
	status := 200
	svc := service.NewMonitorCheckService(
		newMemoryMonitors(monitor),
		newMemoryCheckResults(),
		stubChecker{result: models.CheckResult{
			StatusCode:     &status,
			ResponseTimeMs: 15,
			Success:        true,
			CheckedAt:      time.Now().UTC(),
		}},
		nil,
		nil,
	)
	svc.SetMetrics(m)
	w := New(svc)
	w.metrics = m

	if err := w.HandleJob(context.Background(), queue.NewMonitorCheckJob(monitor.ID)); err != nil {
		t.Fatal(err)
	}

	if got := counterValue(t, m, "pulse_worker_jobs_processed_total", prometheus.Labels{"result": "success"}); got != 1 {
		t.Fatalf("processed = %v, want 1", got)
	}
	if got := counterValue(t, m, "pulse_monitor_checks_total", prometheus.Labels{"result": "success"}); got != 1 {
		t.Fatalf("checks = %v, want 1", got)
	}
}

func TestHandleJobDuplicateDoesNotCountSecondCheck(t *testing.T) {
	m := metrics.New("worker")

	monitor := models.Monitor{
		ID:                 uuid.New(),
		Name:               "test",
		URL:                "https://example.com/health",
		HTTPMethod:         "GET",
		TimeoutSeconds:     2,
		ExpectedStatusCode: 200,
		IsActive:           true,
	}
	status := 200
	svc := service.NewMonitorCheckService(
		newMemoryMonitors(monitor),
		newMemoryCheckResults(),
		stubChecker{result: models.CheckResult{
			StatusCode:     &status,
			ResponseTimeMs: 15,
			Success:        true,
			CheckedAt:      time.Now().UTC(),
		}},
		nil,
		nil,
	)
	job := queue.NewMonitorCheckJob(monitor.ID)
	w := instrumentedWorker(svc, m)
	if err := w.HandleJob(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	if err := w.HandleJob(context.Background(), job); err != nil {
		t.Fatal(err)
	}

	if got := counterValue(t, m, "pulse_monitor_checks_total", prometheus.Labels{"result": "success"}); got != 1 {
		t.Fatalf("checks = %v, want 1 after duplicate delivery", got)
	}
	if got := counterValue(t, m, "pulse_worker_jobs_processed_total", prometheus.Labels{"result": "success"}); got != 2 {
		t.Fatalf("processed = %v, want 2 handled jobs", got)
	}
}

func TestHandleJobInternalFailureIncrementsFailed(t *testing.T) {
	m := metrics.New("worker")

	monitor := models.Monitor{
		ID:                 uuid.New(),
		Name:               "test",
		URL:                "https://example.com/health",
		HTTPMethod:         "GET",
		TimeoutSeconds:     2,
		ExpectedStatusCode: 200,
		IsActive:           true,
	}
	status := 200
	svc := service.NewMonitorCheckService(
		newMemoryMonitors(monitor),
		failingCheckResults{err: errors.New("postgres unavailable")},
		stubChecker{result: models.CheckResult{
			StatusCode:     &status,
			ResponseTimeMs: 15,
			Success:        true,
			CheckedAt:      time.Now().UTC(),
		}},
		nil,
		nil,
	)

	if err := instrumentedWorker(svc, m).HandleJob(context.Background(), queue.NewMonitorCheckJob(monitor.ID)); err == nil {
		t.Fatal("expected internal error")
	}
	if got := counterValue(t, m, "pulse_worker_jobs_failed_total", nil); got != 1 {
		t.Fatalf("failed = %v, want 1", got)
	}
	if got := counterValue(t, m, "pulse_monitor_checks_total", prometheus.Labels{"result": "success"}); got != 0 {
		t.Fatalf("checks = %v, want 0", got)
	}
}

func TestHandleJobUnknownMonitorIsSkipped(t *testing.T) {
	m := metrics.New("worker")

	svc := service.NewMonitorCheckService(newMemoryMonitors(), newMemoryCheckResults(), stubChecker{}, nil, nil)
	if err := instrumentedWorker(svc, m).HandleJob(context.Background(), queue.NewMonitorCheckJob(uuid.New())); err != nil {
		t.Fatal(err)
	}
	if got := counterValue(t, m, "pulse_worker_jobs_processed_total", prometheus.Labels{"result": "skipped"}); got != 1 {
		t.Fatalf("skipped = %v, want 1", got)
	}
}

func TestHandleJobFailedHTTPCheckIncrementsFailure(t *testing.T) {
	m := metrics.New("worker")

	monitor := models.Monitor{
		ID:                 uuid.New(),
		Name:               "test",
		URL:                "https://example.com/health",
		HTTPMethod:         "GET",
		TimeoutSeconds:     2,
		ExpectedStatusCode: 200,
		IsActive:           true,
	}
	status := 500
	message := "unexpected status code"
	svc := service.NewMonitorCheckService(
		newMemoryMonitors(monitor),
		newMemoryCheckResults(),
		stubChecker{result: models.CheckResult{
			StatusCode:     &status,
			ResponseTimeMs: 20,
			Success:        false,
			ErrorMessage:   &message,
			CheckedAt:      time.Now().UTC(),
		}},
		nil,
		nil,
	)
	if err := instrumentedWorker(svc, m).HandleJob(context.Background(), queue.NewMonitorCheckJob(monitor.ID)); err != nil {
		t.Fatal(err)
	}
	if got := counterValue(t, m, "pulse_monitor_check_failures_total", nil); got != 1 {
		t.Fatalf("failures = %v, want 1", got)
	}
}

func instrumentedWorker(svc *service.MonitorCheckService, m *metrics.Metrics) *Worker {
	svc.SetMetrics(m)
	w := New(svc)
	w.metrics = m
	return w
}

func counterValue(t *testing.T, m *metrics.Metrics, name string, labels prometheus.Labels) float64 {
	t.Helper()
	families, err := m.Gatherer().Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.Metric {
			if !labelsMatch(metric.GetLabel(), labels) {
				continue
			}
			if metric.Counter != nil {
				return metric.Counter.GetValue()
			}
		}
	}
	return 0
}

func labelsMatch(got []*dto.LabelPair, want prometheus.Labels) bool {
	if len(want) == 0 {
		return true
	}
	values := map[string]string{}
	for _, label := range got {
		values[label.GetName()] = label.GetValue()
	}
	for key, value := range want {
		if values[key] != value {
			return false
		}
	}
	return true
}
