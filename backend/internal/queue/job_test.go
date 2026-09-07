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
	if decision.RetryJob != nil || decision.DeadLetter != nil {
		t.Fatalf("malformed jobs should be dropped, not retried: %+v", decision)
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
	if decision.RetryJob != nil || decision.DeadLetter != nil {
		t.Fatalf("successful jobs should not be retried or dead-lettered: %+v", decision)
	}
}

func TestHandleDeliveryInternalFailureRetriesWithIncrementedAttempt(t *testing.T) {
	job := NewMonitorCheckJob(uuid.New())
	job.Attempt = 1
	body, _ := EncodeJob(job)

	decision := HandleDelivery(context.Background(), body, func(context.Context, MonitorCheckJob) error {
		return context.DeadlineExceeded
	})
	if !decision.Ack || decision.Requeue {
		t.Fatalf("decision = %+v, want ack after publishing a delayed retry", decision)
	}
	if decision.RetryJob == nil {
		t.Fatal("expected a retry job")
	}
	if decision.RetryJob.JobID != job.JobID {
		t.Fatalf("retry job_id = %s, want %s", decision.RetryJob.JobID, job.JobID)
	}
	if decision.RetryJob.Attempt != 2 {
		t.Fatalf("retry attempt = %d, want 2", decision.RetryJob.Attempt)
	}
	if decision.RetryKey != RetryQueueName(1) {
		t.Fatalf("retry key = %s, want %s", decision.RetryKey, RetryQueueName(1))
	}
	if decision.DeadLetter != nil {
		t.Fatal("attempt 1 should not go to the DLQ")
	}
}

func TestHandleDeliverySecondFailureUsesLongerRetry(t *testing.T) {
	job := NewMonitorCheckJob(uuid.New())
	job.Attempt = 2
	body, _ := EncodeJob(job)

	decision := HandleDelivery(context.Background(), body, func(context.Context, MonitorCheckJob) error {
		return context.DeadlineExceeded
	})
	if decision.RetryJob == nil || decision.RetryJob.Attempt != 3 {
		t.Fatalf("retry job = %+v, want attempt 3", decision.RetryJob)
	}
	if decision.RetryKey != RetryQueueName(2) {
		t.Fatalf("retry key = %s, want %s", decision.RetryKey, RetryQueueName(2))
	}
}

func TestHandleDeliveryMaxAttemptsGoesToDLQ(t *testing.T) {
	job := NewMonitorCheckJob(uuid.New())
	job.Attempt = MaxAttempts
	body, _ := EncodeJob(job)

	decision := HandleDelivery(context.Background(), body, func(context.Context, MonitorCheckJob) error {
		return context.DeadlineExceeded
	})
	if !decision.Ack || decision.Requeue {
		t.Fatalf("decision = %+v, want ack after publishing to the DLQ", decision)
	}
	if decision.RetryJob != nil {
		t.Fatal("max attempts should not publish another retry")
	}
	if decision.DeadLetter == nil {
		t.Fatal("expected a dead-letter job")
	}
	if decision.DeadLetter.JobID != job.JobID {
		t.Fatalf("dlq job_id = %s, want %s", decision.DeadLetter.JobID, job.JobID)
	}
	if decision.DeadLetter.Attempt != MaxAttempts {
		t.Fatalf("dlq attempt = %d, want %d", decision.DeadLetter.Attempt, MaxAttempts)
	}
}
