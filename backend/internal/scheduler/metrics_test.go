package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sreenidhbonagiri/pulse/backend/internal/metrics"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
)

func TestTickIncrementsEnqueuedCounter(t *testing.T) {
	m := metrics.New("scheduler")

	now := time.Now().UTC()
	due := newTestMonitor(true, ptrTime(now.Add(-time.Second)))
	s := New(newMemoryMonitors(due), queue.NewMemoryPublisher())
	s.metrics = m
	s.Tick(context.Background(), now)

	if got := schedulerCounter(t, m, "pulse_scheduler_jobs_enqueued_total"); got != 1 {
		t.Fatalf("enqueued = %v, want 1", got)
	}
	if got := schedulerCounter(t, m, "pulse_scheduler_enqueue_failures_total"); got != 0 {
		t.Fatalf("failures = %v, want 0", got)
	}
}

func TestTickPublishFailureIncrementsEnqueueFailures(t *testing.T) {
	m := metrics.New("scheduler")

	now := time.Now().UTC()
	due := newTestMonitor(true, ptrTime(now.Add(-time.Second)))
	publisher := queue.NewMemoryPublisher()
	publisher.Err = errors.New("rabbitmq down")
	s := New(newMemoryMonitors(due), publisher)
	s.metrics = m
	s.Tick(context.Background(), now)

	if got := schedulerCounter(t, m, "pulse_scheduler_jobs_enqueued_total"); got != 0 {
		t.Fatalf("enqueued = %v, want 0", got)
	}
	if got := schedulerCounter(t, m, "pulse_scheduler_enqueue_failures_total"); got < 1 {
		t.Fatalf("failures = %v, want at least 1", got)
	}
}

func schedulerCounter(t *testing.T, m *metrics.Metrics, name string) float64 {
	t.Helper()
	families, err := m.Gatherer().Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		var total float64
		for _, metric := range family.Metric {
			if metric.Counter != nil {
				total += metric.Counter.GetValue()
			}
			if metric.Histogram != nil {
				total += float64(metric.Histogram.GetSampleCount())
			}
		}
		return total
	}
	return 0
}
