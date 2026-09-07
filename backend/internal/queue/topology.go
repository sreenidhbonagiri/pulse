package queue

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func declareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(JobsExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange %s: %w", JobsExchange, err)
	}

	if _, err := ch.QueueDeclare(MonitorChecksQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare queue %s: %w", MonitorChecksQueue, err)
	}
	if err := ch.QueueBind(MonitorChecksQueue, MonitorChecksQueue, JobsExchange, false, nil); err != nil {
		return fmt.Errorf("bind queue %s: %w", MonitorChecksQueue, err)
	}

	for i := 1; i <= retryQueueCount(); i++ {
		name := RetryQueueName(i)
		ttl := RetryTTLs[i-1]
		if _, err := ch.QueueDeclare(name, true, false, false, false, amqp.Table{
			"x-message-ttl":             int32(ttl.Milliseconds()),
			"x-dead-letter-exchange":    JobsExchange,
			"x-dead-letter-routing-key": MonitorChecksQueue,
		}); err != nil {
			return fmt.Errorf("declare retry queue %s: %w", name, err)
		}
		if err := ch.QueueBind(name, name, JobsExchange, false, nil); err != nil {
			return fmt.Errorf("bind retry queue %s: %w", name, err)
		}
	}

	if _, err := ch.QueueDeclare(DeadLetterQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare queue %s: %w", DeadLetterQueue, err)
	}
	if err := ch.QueueBind(DeadLetterQueue, DeadLetterQueue, JobsExchange, false, nil); err != nil {
		return fmt.Errorf("bind queue %s: %w", DeadLetterQueue, err)
	}

	return nil
}
