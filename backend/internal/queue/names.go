package queue

import (
	"strconv"
	"time"
)

const (
	JobsExchange       = "pulse.jobs"
	MonitorChecksQueue = "monitor.checks"
	DeadLetterQueue    = "monitor.checks.dlq"
	MaxAttempts        = 3
)

// RetryTTLs are delays before the 2nd and 3rd attempts (short, then longer).
var RetryTTLs = []time.Duration{
	2 * time.Second,
	8 * time.Second,
}

func RetryQueueName(index int) string {
	return MonitorChecksQueue + ".retry." + strconv.Itoa(index)
}

func retryQueueCount() int {
	count := MaxAttempts - 1
	if count > len(RetryTTLs) {
		return len(RetryTTLs)
	}
	if count < 1 {
		return 1
	}
	return count
}
