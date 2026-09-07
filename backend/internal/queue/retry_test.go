package queue

import (
	"testing"

	"github.com/google/uuid"
)

func TestRouteAfterFailureIncrementsAttempt(t *testing.T) {
	job := NewMonitorCheckJob(uuid.New())
	job.Attempt = 1

	route := RouteAfterFailure(job)
	if route.Kind != RouteRetry {
		t.Fatalf("kind = %v, want retry", route.Kind)
	}
	if route.Job.JobID != job.JobID {
		t.Fatalf("job_id changed: %s", route.Job.JobID)
	}
	if route.Job.Attempt != 2 {
		t.Fatalf("attempt = %d, want 2", route.Job.Attempt)
	}
	if route.RoutingKey != RetryQueueName(1) {
		t.Fatalf("routing key = %s", route.RoutingKey)
	}
}

func TestRouteAfterFailureMaxAttemptsGoesToDLQ(t *testing.T) {
	job := NewMonitorCheckJob(uuid.New())
	job.Attempt = MaxAttempts

	route := RouteAfterFailure(job)
	if route.Kind != RouteDeadLetter {
		t.Fatalf("kind = %v, want dead letter", route.Kind)
	}
	if route.Job.JobID != job.JobID {
		t.Fatalf("job_id changed: %s", route.Job.JobID)
	}
	if route.Job.Attempt != MaxAttempts {
		t.Fatalf("attempt = %d, want %d", route.Job.Attempt, MaxAttempts)
	}
	if route.RoutingKey != DeadLetterQueue {
		t.Fatalf("routing key = %s, want %s", route.RoutingKey, DeadLetterQueue)
	}
}

func TestRetryQueueNames(t *testing.T) {
	if MonitorChecksQueue != "monitor.checks" {
		t.Fatalf("main queue = %s", MonitorChecksQueue)
	}
	if DeadLetterQueue != "monitor.checks.dlq" {
		t.Fatalf("dlq = %s", DeadLetterQueue)
	}
	if RetryQueueName(1) != "monitor.checks.retry.1" {
		t.Fatalf("retry 1 = %s", RetryQueueName(1))
	}
	if retryQueueCount() != 2 {
		t.Fatalf("retry queue count = %d, want 2", retryQueueCount())
	}
}
