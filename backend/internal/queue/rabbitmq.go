package queue

import (
	"context"
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQ publishes and consumes monitor-check jobs.
type RabbitMQ struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	mu   sync.Mutex
}

func Dial(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}

	if err := declareTopology(ch); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	if err := ch.Qos(1, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("set rabbitmq qos: %w", err)
	}

	return &RabbitMQ{conn: conn, ch: ch}, nil
}

func (r *RabbitMQ) Publish(ctx context.Context, job MonitorCheckJob) error {
	return r.publishTo(ctx, MonitorChecksQueue, job)
}

func (r *RabbitMQ) Consume(ctx context.Context, handler JobHandler) error {
	msgs, err := r.ch.Consume(MonitorChecksQueue, "pulse-worker", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume %s: %w", MonitorChecksQueue, err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-msgs:
			if !ok {
				return nil
			}
			r.handleOne(ctx, delivery, handler)
		}
	}
}

func (r *RabbitMQ) handleOne(ctx context.Context, delivery amqp.Delivery, handler JobHandler) {
	decision := HandleDelivery(ctx, delivery.Body, handler)

	if decision.RetryJob != nil {
		if err := r.publishTo(ctx, decision.RetryKey, *decision.RetryJob); err != nil {
			_ = delivery.Nack(false, true)
			return
		}
	}
	if decision.DeadLetter != nil {
		if err := r.publishTo(ctx, DeadLetterQueue, *decision.DeadLetter); err != nil {
			_ = delivery.Nack(false, true)
			return
		}
	}

	if decision.Ack {
		_ = delivery.Ack(false)
		return
	}
	_ = delivery.Nack(false, decision.Requeue)
}

func (r *RabbitMQ) publishTo(ctx context.Context, routingKey string, job MonitorCheckJob) error {
	body, err := EncodeJob(job)
	if err != nil {
		return fmt.Errorf("encode job: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	err = r.ch.PublishWithContext(ctx, JobsExchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    job.JobID.String(),
		Body:         body,
	})
	if err != nil {
		return fmt.Errorf("publish job %s to %s: %w", job.JobID, routingKey, err)
	}
	return nil
}

func (r *RabbitMQ) Close() error {
	if r.ch != nil {
		_ = r.ch.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}
