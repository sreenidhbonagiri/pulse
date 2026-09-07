package queue

import "context"

// MemoryPublisher stores jobs in memory. Tests use this instead of RabbitMQ.
type MemoryPublisher struct {
	Jobs []MonitorCheckJob
	Err  error
}

func NewMemoryPublisher() *MemoryPublisher {
	return &MemoryPublisher{Jobs: make([]MonitorCheckJob, 0)}
}

func (m *MemoryPublisher) Publish(_ context.Context, job MonitorCheckJob) error {
	if m.Err != nil {
		return m.Err
	}
	m.Jobs = append(m.Jobs, job)
	return nil
}
