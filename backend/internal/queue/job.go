package queue

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// MonitorCheckJob is a request to run one health check in the background.
type MonitorCheckJob struct {
	JobID       uuid.UUID `json:"job_id"`
	MonitorID   uuid.UUID `json:"monitor_id"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Attempt     int       `json:"attempt"`
}

func NewMonitorCheckJob(monitorID uuid.UUID) MonitorCheckJob {
	return MonitorCheckJob{
		JobID:       uuid.New(),
		MonitorID:   monitorID,
		ScheduledAt: time.Now().UTC(),
		Attempt:     1,
	}
}

func EncodeJob(job MonitorCheckJob) ([]byte, error) {
	return json.Marshal(job)
}

func DecodeJob(body []byte) (MonitorCheckJob, error) {
	var job MonitorCheckJob
	if err := json.Unmarshal(body, &job); err != nil {
		return MonitorCheckJob{}, errors.New("invalid job JSON")
	}
	if job.JobID == uuid.Nil || job.MonitorID == uuid.Nil {
		return MonitorCheckJob{}, errors.New("job_id and monitor_id are required")
	}
	return job, nil
}
