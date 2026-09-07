package queue

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEncodeDecodeJob(t *testing.T) {
	original := MonitorCheckJob{
		JobID:       uuid.New(),
		MonitorID:   uuid.New(),
		ScheduledAt: time.Date(2026, 9, 6, 15, 0, 0, 0, time.UTC),
		Attempt:     1,
	}

	body, err := EncodeJob(original)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	decoded, err := DecodeJob(body)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.JobID != original.JobID || decoded.MonitorID != original.MonitorID {
		t.Fatalf("decoded = %+v, want %+v", decoded, original)
	}
	if decoded.Attempt != 1 {
		t.Fatalf("attempt = %d", decoded.Attempt)
	}
	if !decoded.ScheduledAt.Equal(original.ScheduledAt) {
		t.Fatalf("scheduled_at = %s", decoded.ScheduledAt)
	}
}

func TestDecodeJobMalformed(t *testing.T) {
	if _, err := DecodeJob([]byte("not-json")); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if _, err := DecodeJob([]byte(`{"job_id":"","monitor_id":""}`)); err == nil {
		t.Fatal("expected error for missing ids")
	}
}

func TestHandleDeliveryMalformedAcksWithoutRequeue(t *testing.T) {
	called := false
	decision := HandleDelivery(context.Background(), []byte("not-json"), func(context.Context, MonitorCheckJob) error {
		called = true
		return nil
	})
	if called {
		t.Fatal("handler should not run for a malformed message")
	}
	if !decision.Ack || decision.Requeue {
		t.Fatalf("decision = %+v, want ack without requeue", decision)
	}
}

func TestHandleDeliverySuccessAcks(t *testing.T) {
	job := NewMonitorCheckJob(uuid.New())
	body, err := EncodeJob(job)
	if err != nil {
		t.Fatal(err)
	}

	decision := HandleDelivery(context.Background(), body, func(_ context.Context, got MonitorCheckJob) error {
		if got.JobID != job.JobID {
			t.Fatalf("job_id = %s", got.JobID)
		}
		return nil
	})
	if !decision.Ack || decision.Requeue {
		t.Fatalf("decision = %+v", decision)
	}
}

func TestHandleDeliveryHandlerErrorRequeues(t *testing.T) {
	job := NewMonitorCheckJob(uuid.New())
	body, _ := EncodeJob(job)

	decision := HandleDelivery(context.Background(), body, func(context.Context, MonitorCheckJob) error {
		return context.DeadlineExceeded
	})
	if decision.Ack || !decision.Requeue {
		t.Fatalf("decision = %+v, want nack with requeue", decision)
	}
}
