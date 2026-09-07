package queue

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/metrics"
)

func TestHandleDeliveryRetryIncrementsRetryCounter(t *testing.T) {
	m := metrics.New("worker")

	job := NewMonitorCheckJob(uuid.New())
	body, err := EncodeJob(job)
	if err != nil {
		t.Fatal(err)
	}

	decision := HandleDelivery(context.Background(), body, func(context.Context, MonitorCheckJob) error {
		return errors.New("postgres unavailable")
	})
	RecordDeliveryMetrics(m, decision)
	if decision.RetryJob == nil {
		t.Fatal("expected retry routing")
	}
	if got := counterNamed(t, m, "pulse_worker_retries_total"); got != 1 {
		t.Fatalf("retries = %v, want 1", got)
	}
	if got := counterNamed(t, m, "pulse_worker_dlq_total"); got != 0 {
		t.Fatalf("dlq = %v, want 0", got)
	}
}

func TestHandleDeliveryMaxAttemptsIncrementsDLQ(t *testing.T) {
	m := metrics.New("worker")

	job := NewMonitorCheckJob(uuid.New())
	job.Attempt = MaxAttempts
	body, err := EncodeJob(job)
	if err != nil {
		t.Fatal(err)
	}

	decision := HandleDelivery(context.Background(), body, func(context.Context, MonitorCheckJob) error {
		return errors.New("postgres unavailable")
	})
	RecordDeliveryMetrics(m, decision)
	if decision.DeadLetter == nil {
		t.Fatal("expected dead-letter routing")
	}
	if got := counterNamed(t, m, "pulse_worker_dlq_total"); got != 1 {
		t.Fatalf("dlq = %v, want 1", got)
	}
}

func counterNamed(t *testing.T, m *metrics.Metrics, name string) float64 {
	t.Helper()
	families, err := m.Gatherer().Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		if len(family.Metric) == 0 || family.Metric[0].Counter == nil {
			return 0
		}
		return family.Metric[0].Counter.GetValue()
	}
	return 0
}
