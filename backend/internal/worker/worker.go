package worker

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/sreenidhbonagiri/pulse/backend/internal/metrics"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
	"github.com/sreenidhbonagiri/pulse/backend/internal/service"
)

// Worker consumes monitor-check jobs and saves CheckResults.
type Worker struct {
	checks  *service.MonitorCheckService
	metrics *metrics.Metrics
}

func New(checks *service.MonitorCheckService) *Worker {
	return &Worker{checks: checks, metrics: metrics.Default()}
}

func (w *Worker) HandleJob(ctx context.Context, job queue.MonitorCheckJob) error {
	start := time.Now()
	result, err := w.checks.RunCheck(ctx, job.MonitorID, job.JobID)
	duration := time.Since(start)
	if errors.Is(err, repository.ErrNotFound) {
		log.Printf("job_id=%s monitor_id=%s attempt=%d skipped: monitor not found", job.JobID, job.MonitorID, job.Attempt)
		w.metrics.ObserveWorkerJob(metrics.ResultSkipped, duration)
		return nil
	}
	if err != nil {
		log.Printf("job_id=%s monitor_id=%s attempt=%d error=%v", job.JobID, job.MonitorID, job.Attempt, err)
		w.metrics.WorkerFailed(duration)
		return err
	}

	statusCode := "-"
	if result.StatusCode != nil {
		statusCode = strconv.Itoa(*result.StatusCode)
	}

	log.Printf(
		"job_id=%s monitor_id=%s attempt=%d status_code=%s response_time_ms=%d success=%t",
		job.JobID,
		job.MonitorID,
		job.Attempt,
		statusCode,
		result.ResponseTimeMs,
		result.Success,
	)
	w.metrics.ObserveWorkerJob(metrics.ResultSuccess, duration)
	return nil
}
