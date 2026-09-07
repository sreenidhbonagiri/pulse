package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

const (
	DefaultPollInterval = 3 * time.Second
	DefaultBatchSize    = 25
)

// Scheduler finds due monitors and enqueues check jobs.
type Scheduler struct {
	monitors  repository.MonitorRepository
	publisher queue.Publisher
	interval  time.Duration
	batchSize int
}

func New(monitors repository.MonitorRepository, publisher queue.Publisher) *Scheduler {
	return &Scheduler{
		monitors:  monitors,
		publisher: publisher,
		interval:  DefaultPollInterval,
		batchSize: DefaultBatchSize,
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.Tick(ctx, time.Now().UTC())

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.Tick(ctx, time.Now().UTC())
		}
	}
}

func (s *Scheduler) Tick(ctx context.Context, now time.Time) {
	for i := 0; i < s.batchSize; i++ {
		if ctx.Err() != nil {
			return
		}

		monitor, err := s.monitors.ClaimDue(ctx, now, func(monitor models.Monitor) error {
			job := queue.NewMonitorCheckJob(monitor.ID)
			if err := s.publisher.Publish(ctx, job); err != nil {
				return err
			}
			log.Printf("scheduled monitor_id=%s job_id=%s", monitor.ID, job.JobID)
			return nil
		})
		if err != nil {
			log.Printf("scheduler failed to enqueue a due monitor: %v", err)
			continue
		}
		if monitor == nil {
			return
		}
	}
}
