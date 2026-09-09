package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/sreenidhbonagiri/pulse/backend/internal/metrics"
)

// SQS publishes and consumes monitor-check jobs using Amazon SQS.
type SQS struct {
	client   *sqs.Client
	queueURL string
	dlqURL   string
}

// NewSQS creates an SQS-backed queue using the normal AWS credential chain.
// In ECS, credentials will come from the task IAM role automatically.
func NewSQS(ctx context.Context, queueURL, dlqURL string) (*SQS, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	return &SQS{
		client:   sqs.NewFromConfig(cfg),
		queueURL: queueURL,
		dlqURL:   dlqURL,
	}, nil
}

// Publish sends a new monitor-check job to the primary queue.
func (s *SQS) Publish(ctx context.Context, job MonitorCheckJob) error {
	return s.publishTo(ctx, s.queueURL, job, 0)
}

// Consume long-polls SQS and processes jobs until the context is canceled.
func (s *SQS) Consume(ctx context.Context, handler JobHandler) error {
	for {
		if ctx.Err() != nil {
			return nil
		}

		result, err := s.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(s.queueURL),
			MaxNumberOfMessages: 1,
			WaitTimeSeconds:     20,
			VisibilityTimeout:   30,
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("receive SQS message: %w", err)
		}

		for _, message := range result.Messages {
			if err := s.handleOne(ctx, message.Body, message.ReceiptHandle, handler); err != nil {
				return err
			}
		}
	}
}

func (s *SQS) handleOne(
	ctx context.Context,
	body *string,
	receiptHandle *string,
	handler JobHandler,
) error {
	if body == nil || receiptHandle == nil {
		return nil
	}

	decision := HandleDelivery(ctx, []byte(*body), handler)
	RecordDeliveryMetrics(metrics.Default(), decision)

	// A retry is published back to the main queue with the same delay
	// used by the RabbitMQ retry queues.
	if decision.RetryJob != nil {
		delay := retryDelayForAttempt(decision.RetryJob.Attempt)

		if err := s.publishTo(
			ctx,
			s.queueURL,
			*decision.RetryJob,
			delay,
		); err != nil {
			// Do not delete the original message. SQS will make it visible
			// again after the visibility timeout.
			return err
		}
	}

	if decision.DeadLetter != nil {
		if err := s.publishTo(
			ctx,
			s.dlqURL,
			*decision.DeadLetter,
			0,
		); err != nil {
			return err
		}
	}

	if decision.Ack {
		_, err := s.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
			QueueUrl:      aws.String(s.queueURL),
			ReceiptHandle: receiptHandle,
		})
		if err != nil {
			return fmt.Errorf("delete SQS message: %w", err)
		}
	}

	return nil
}

func (s *SQS) publishTo(
	ctx context.Context,
	queueURL string,
	job MonitorCheckJob,
	delay time.Duration,
) error {
	body, err := EncodeJob(job)
	if err != nil {
		return fmt.Errorf("encode job: %w", err)
	}

	delaySeconds := int32(delay / time.Second)

	_, err = s.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:     aws.String(queueURL),
		MessageBody:  aws.String(string(body)),
		DelaySeconds: delaySeconds,
	})
	if err != nil {
		return fmt.Errorf("publish SQS job %s: %w", job.JobID, err)
	}

	return nil
}

// retryDelayForAttempt preserves the RabbitMQ retry timing:
// attempt 2 waits 2 seconds and attempt 3 waits 8 seconds.
func retryDelayForAttempt(attempt int) time.Duration {
	index := attempt - 2

	if index < 0 {
		index = 0
	}
	if index >= len(RetryTTLs) {
		index = len(RetryTTLs) - 1
	}

	return RetryTTLs[index]
}

// Close exists so RabbitMQ and SQS can be handled similarly.
// The AWS SQS client does not require an explicit close.
func (s *SQS) Close() error {
	return nil
}
