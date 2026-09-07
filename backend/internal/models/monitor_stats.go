package models

// MonitorStats is a dashboard summary for one monitor.
type MonitorStats struct {
	CurrentStatus    string  `json:"current_status"`
	UptimePercentage float64 `json:"uptime_percentage"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
	P50LatencyMs     float64 `json:"p50_latency_ms"`
	P95LatencyMs     float64 `json:"p95_latency_ms"`
	P99LatencyMs     float64 `json:"p99_latency_ms"`
	TotalChecks      int     `json:"total_checks"`
	FailedChecks     int     `json:"failed_checks"`
	IncidentCount    int     `json:"incident_count"`
	ActiveIncident   bool    `json:"active_incident"`
}

const (
	MonitorStatusUp      = "up"
	MonitorStatusDown    = "down"
	MonitorStatusUnknown = "unknown"
)
