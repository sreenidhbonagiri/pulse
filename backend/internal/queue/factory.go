package queue

import (
	"context"
	"fmt"
	"strings"
)

type Client interface {
	Publisher
	Consume(ctx context.Context, handler JobHandler) error
	Close() error
}

type Config struct {
	Provider    string
	RabbitMQURL string
	SQSQueueURL string
	SQSDLQURL   string
}

func New(ctx context.Context, cfg Config) (Client, error) {
	switch strings.ToLower(cfg.Provider) {
	case "", "rabbitmq":
		return Dial(cfg.RabbitMQURL)

	case "sqs":
		if cfg.SQSQueueURL == "" {
			return nil, fmt.Errorf("SQS_QUEUE_URL is required when QUEUE_PROVIDER=sqs")
		}
		if cfg.SQSDLQURL == "" {
			return nil, fmt.Errorf("SQS_DLQ_URL is required when QUEUE_PROVIDER=sqs")
		}
		return NewSQS(ctx, cfg.SQSQueueURL, cfg.SQSDLQURL)

	default:
		return nil, fmt.Errorf("unsupported queue provider %q", cfg.Provider)
	}
}
