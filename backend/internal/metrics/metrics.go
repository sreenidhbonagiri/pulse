package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	ResultSuccess   = "success"
	ResultFailure   = "failure"
	ResultDuplicate = "duplicate"
	ResultSkipped   = "skipped"
)

// Metrics holds Pulse's Prometheus collectors. Methods are nil-safe so tests
// and processes can run without a registry.
type Metrics struct {
	registry prometheus.Registerer
	gatherer prometheus.Gatherer

	HTTPRequests          *prometheus.CounterVec
	HTTPRequestDuration   *prometheus.HistogramVec
	HTTPRequestsInFlight  prometheus.Gauge
	MonitorChecks         *prometheus.CounterVec
	MonitorCheckDuration  prometheus.Histogram
	MonitorCheckFailures  prometheus.Counter
	WorkerJobsProcessed   *prometheus.CounterVec
	WorkerJobsFailed      prometheus.Counter
	WorkerJobDuration     prometheus.Histogram
	WorkerRetries         prometheus.Counter
	WorkerDLQ             prometheus.Counter
	SchedulerJobsEnqueued prometheus.Counter
	SchedulerEnqueueFail  prometheus.Counter
	SchedulerCycle        prometheus.Histogram
	ActiveIncidents       prometheus.Gauge
	DatabaseErrors        prometheus.Counter
	RedisErrors           prometheus.Counter
	RabbitMQErrors        prometheus.Counter
}

var (
	globalMu sync.RWMutex
	global   *Metrics
)

func Default() *Metrics {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return global
}

func SetDefault(m *Metrics) func() {
	globalMu.Lock()
	prev := global
	global = m
	globalMu.Unlock()
	return func() {
		globalMu.Lock()
		global = prev
		globalMu.Unlock()
	}
}

func Init(service string) *Metrics {
	m := New(service)
	SetDefault(m)
	return m
}

func New(service string) *Metrics {
	reg := prometheus.NewRegistry()
	return newWith(service, reg, reg)
}

func newWith(service string, registerer prometheus.Registerer, gatherer prometheus.Gatherer) *Metrics {
	if service == "" {
		service = "unknown"
	}

	m := &Metrics{
		registry: registerer,
		gatherer: gatherer,
		HTTPRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "pulse_http_requests_total",
			Help: "HTTP requests handled by the Pulse API.",
		}, []string{"method", "route", "status_class"}),
		HTTPRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "pulse_http_request_duration_seconds",
			Help:    "Duration of Pulse API HTTP requests.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}, []string{"method", "route", "status_class"}),
		HTTPRequestsInFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "pulse_http_requests_in_flight",
			Help: "API requests currently being handled.",
		}),
		MonitorChecks: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "pulse_monitor_checks_total",
			Help: "Completed monitor HTTP checks, excluding duplicate deliveries.",
		}, []string{"result"}),
		MonitorCheckDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "pulse_monitor_check_duration_seconds",
			Help:    "Duration of monitor HTTP checks.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		}),
		MonitorCheckFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "pulse_monitor_check_failures_total",
			Help: "Monitor HTTP checks that did not meet the expected result.",
		}),
		WorkerJobsProcessed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "pulse_worker_jobs_processed_total",
			Help: "Monitor-check jobs the worker finished without an internal error.",
		}, []string{"result"}),
		WorkerJobsFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "pulse_worker_jobs_failed_total",
			Help: "Monitor-check jobs that failed internally and may be retried.",
		}),
		WorkerJobDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "pulse_worker_processing_duration_seconds",
			Help:    "Time the worker spent handling one job.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		}),
		WorkerRetries: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "pulse_worker_retries_total",
			Help: "Jobs routed to a retry queue after an internal failure.",
		}),
		WorkerDLQ: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "pulse_worker_dlq_total",
			Help: "Jobs routed to the dead-letter queue after exhausting retries.",
		}),
		SchedulerJobsEnqueued: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "pulse_scheduler_jobs_enqueued_total",
			Help: "Check jobs the scheduler published successfully.",
		}),
		SchedulerEnqueueFail: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "pulse_scheduler_enqueue_failures_total",
			Help: "Scheduler attempts that failed to claim or publish a due monitor.",
		}),
		SchedulerCycle: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "pulse_scheduler_cycle_duration_seconds",
			Help:    "Duration of one scheduler poll cycle.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		}),
		ActiveIncidents: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "pulse_active_incidents",
			Help: "Open incidents across all monitors.",
		}),
		DatabaseErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name:        "pulse_database_errors_total",
			Help:        "Unexpected PostgreSQL errors.",
			ConstLabels: prometheus.Labels{"service": service},
		}),
		RedisErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name:        "pulse_redis_errors_total",
			Help:        "Redis command errors.",
			ConstLabels: prometheus.Labels{"service": service},
		}),
		RabbitMQErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name:        "pulse_rabbitmq_errors_total",
			Help:        "RabbitMQ connection or publish errors.",
			ConstLabels: prometheus.Labels{"service": service},
		}),
	}

	registerer.MustRegister(
		m.HTTPRequests,
		m.HTTPRequestDuration,
		m.HTTPRequestsInFlight,
		m.MonitorChecks,
		m.MonitorCheckDuration,
		m.MonitorCheckFailures,
		m.WorkerJobsProcessed,
		m.WorkerJobsFailed,
		m.WorkerJobDuration,
		m.WorkerRetries,
		m.WorkerDLQ,
		m.SchedulerJobsEnqueued,
		m.SchedulerEnqueueFail,
		m.SchedulerCycle,
		m.ActiveIncidents,
		m.DatabaseErrors,
		m.RedisErrors,
		m.RabbitMQErrors,
	)
	return m
}

func (m *Metrics) Gatherer() prometheus.Gatherer {
	if m == nil {
		return nil
	}
	return m.gatherer
}

func (m *Metrics) ObserveHTTP(method, route, statusClass string, duration time.Duration) {
	if m == nil {
		return
	}
	m.HTTPRequests.WithLabelValues(method, route, statusClass).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, route, statusClass).Observe(duration.Seconds())
}

func (m *Metrics) HTTPInFlightInc() {
	if m == nil {
		return
	}
	m.HTTPRequestsInFlight.Inc()
}

func (m *Metrics) HTTPInFlightDec() {
	if m == nil {
		return
	}
	m.HTTPRequestsInFlight.Dec()
}

func (m *Metrics) ObserveCheck(success bool, duration time.Duration) {
	if m == nil {
		return
	}
	result := ResultSuccess
	if !success {
		result = ResultFailure
		m.MonitorCheckFailures.Inc()
	}
	m.MonitorChecks.WithLabelValues(result).Inc()
	m.MonitorCheckDuration.Observe(duration.Seconds())
}

func (m *Metrics) ObserveWorkerJob(result string, duration time.Duration) {
	if m == nil {
		return
	}
	m.WorkerJobsProcessed.WithLabelValues(result).Inc()
	m.WorkerJobDuration.Observe(duration.Seconds())
}

func (m *Metrics) WorkerFailed(duration time.Duration) {
	if m == nil {
		return
	}
	m.WorkerJobsFailed.Inc()
	m.WorkerJobDuration.Observe(duration.Seconds())
}

func (m *Metrics) IncWorkerRetry() {
	if m == nil {
		return
	}
	m.WorkerRetries.Inc()
}

func (m *Metrics) IncWorkerDLQ() {
	if m == nil {
		return
	}
	m.WorkerDLQ.Inc()
}

func (m *Metrics) IncSchedulerEnqueued() {
	if m == nil {
		return
	}
	m.SchedulerJobsEnqueued.Inc()
}

func (m *Metrics) IncSchedulerEnqueueFailure() {
	if m == nil {
		return
	}
	m.SchedulerEnqueueFail.Inc()
}

func (m *Metrics) ObserveSchedulerCycle(duration time.Duration) {
	if m == nil {
		return
	}
	m.SchedulerCycle.Observe(duration.Seconds())
}

func (m *Metrics) SetActiveIncidents(count float64) {
	if m == nil {
		return
	}
	m.ActiveIncidents.Set(count)
}

func (m *Metrics) IncActiveIncidents() {
	if m == nil {
		return
	}
	m.ActiveIncidents.Inc()
}

func (m *Metrics) DecActiveIncidents() {
	if m == nil {
		return
	}
	m.ActiveIncidents.Dec()
}

func (m *Metrics) IncDatabaseErrors() {
	if m == nil {
		return
	}
	m.DatabaseErrors.Inc()
}

func (m *Metrics) IncRedisErrors() {
	if m == nil {
		return
	}
	m.RedisErrors.Inc()
}

func (m *Metrics) IncRabbitMQErrors() {
	if m == nil {
		return
	}
	m.RabbitMQErrors.Inc()
}

func DatabaseError() {
	Default().IncDatabaseErrors()
}

func RedisError() {
	Default().IncRedisErrors()
}

func RabbitMQError() {
	Default().IncRabbitMQErrors()
}
