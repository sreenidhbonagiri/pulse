package monitoring

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

const maxResponseBodyBytes = 512 * 1024

// Checker runs one HTTP health check for a Monitor.
// Reuse a Checker so Pulse shares one HTTP client instead of creating a new
// client on every check.
type Checker struct {
	client *http.Client
}

func NewChecker(client *http.Client) *Checker {
	if client == nil {
		client = &http.Client{}
	}
	return &Checker{client: client}
}

func (c *Checker) Check(ctx context.Context, monitor models.Monitor) models.CheckResult {
	checkedAt := time.Now().UTC()
	result := models.CheckResult{
		ID:        uuid.New(),
		MonitorID: monitor.ID,
		CheckedAt: checkedAt,
	}

	timeout := time.Duration(monitor.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, monitor.HTTPMethod, monitor.URL, nil)
	if err != nil {
		result.ErrorMessage = strPtr("invalid request: " + err.Error())
		return result
	}
	req.Header.Set("User-Agent", "PulseMonitor/1.0")

	start := time.Now()
	resp, err := c.client.Do(req)
	result.ResponseTimeMs = int(time.Since(start).Milliseconds())
	if err != nil {
		result.ErrorMessage = strPtr(classifyRequestError(err))
		return result
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBodyBytes))
	result.ResponseTimeMs = int(time.Since(start).Milliseconds())

	statusCode := resp.StatusCode
	result.StatusCode = &statusCode

	if statusCode != monitor.ExpectedStatusCode {
		result.ErrorMessage = strPtr(
			"unexpected status code: got " + itoa(statusCode) + ", want " + itoa(monitor.ExpectedStatusCode),
		)
		return result
	}

	result.Success = true
	return result
}

func strPtr(value string) *string {
	return &value
}
