package queue

import "fmt"

type RouteKind int

const (
	RouteNone RouteKind = iota
	RouteRetry
	RouteDeadLetter
)

// FailureRoute says where a failed job should go next.
type FailureRoute struct {
	Kind       RouteKind
	Job        MonitorCheckJob
	RoutingKey string
}

// RouteAfterFailure increments attempt and chooses a retry queue or the DLQ.
// The original job_id is preserved.
func RouteAfterFailure(job MonitorCheckJob) FailureRoute {
	if job.Attempt < 1 {
		job.Attempt = 1
	}

	if job.Attempt >= MaxAttempts {
		return FailureRoute{
			Kind:       RouteDeadLetter,
			Job:        job,
			RoutingKey: DeadLetterQueue,
		}
	}

	next := job
	next.Attempt++
	retryIndex := job.Attempt
	if retryIndex > retryQueueCount() {
		retryIndex = retryQueueCount()
	}

	return FailureRoute{
		Kind:       RouteRetry,
		Job:        next,
		RoutingKey: RetryQueueName(retryIndex),
	}
}

func (r FailureRoute) String() string {
	switch r.Kind {
	case RouteRetry:
		return fmt.Sprintf("retry attempt=%d queue=%s", r.Job.Attempt, r.RoutingKey)
	case RouteDeadLetter:
		return fmt.Sprintf("dead-letter attempt=%d queue=%s", r.Job.Attempt, r.RoutingKey)
	default:
		return "none"
	}
}
