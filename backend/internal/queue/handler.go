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

// DeliveryDecision tells the consumer whether to ack or requeue a message.
type DeliveryDecision struct {
	Ack     bool
	Requeue bool
}

// HandleDelivery decodes a raw queue message without crashing on bad JSON.
// Ack the message only after the handler succeeds. Malformed jobs are dropped.
func HandleDelivery(ctx context.Context, body []byte, handler JobHandler) DeliveryDecision {
	job, err := DecodeJob(body)
	if err != nil {
		log.Printf("skipping malformed monitor-check job: %v", err)
		return DeliveryDecision{Ack: true, Requeue: false}
	}

	if err := handler(ctx, job); err != nil {
		log.Printf("monitor-check job failed job_id=%s monitor_id=%s: %v", job.JobID, job.MonitorID, err)
		return DeliveryDecision{Ack: false, Requeue: true}
	}

	return DeliveryDecision{Ack: true, Requeue: false}
}
