package queue

import (
	"context"
	"log"
)

// Publisher sends monitor-check jobs to the queue.
type Publisher interface {
	Publish(ctx context.Context, job MonitorCheckJob) error
}

// JobHandler processes one decoded job.
type JobHandler func(ctx context.Context, job MonitorCheckJob) error

// DeliveryDecision tells the consumer whether to ack and where to route failures.
type DeliveryDecision struct {
	Ack        bool
	Requeue    bool
	RetryJob   *MonitorCheckJob
	RetryKey   string
	DeadLetter *MonitorCheckJob
}

// HandleDelivery decodes a raw queue message without crashing on bad JSON.
func HandleDelivery(ctx context.Context, body []byte, handler JobHandler) DeliveryDecision {
	job, err := DecodeJob(body)
	if err != nil {
		log.Printf("skipping malformed monitor-check job: %v", err)
		return DeliveryDecision{Ack: true}
	}

	if err := handler(ctx, job); err != nil {
		route := RouteAfterFailure(job)
		log.Printf(
			"monitor-check job failed job_id=%s monitor_id=%s attempt=%d route=%s error=%v",
			job.JobID,
			job.MonitorID,
			job.Attempt,
			route,
			err,
		)
		switch route.Kind {
		case RouteRetry:
			next := route.Job
			return DeliveryDecision{Ack: true, RetryJob: &next, RetryKey: route.RoutingKey}
		case RouteDeadLetter:
			dead := route.Job
			return DeliveryDecision{Ack: true, DeadLetter: &dead}
		default:
			return DeliveryDecision{Ack: true}
		}
	}

	return DeliveryDecision{Ack: true}
}
