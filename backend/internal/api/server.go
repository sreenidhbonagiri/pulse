package api

import (
	"net/http"

	"github.com/sreenidhbonagiri/pulse/backend/internal/cache"
	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
	"github.com/sreenidhbonagiri/pulse/backend/internal/metrics"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
	"github.com/sreenidhbonagiri/pulse/backend/internal/service"
)

type Server struct {
	addr        string
	corsOrigins string
	metrics     *metrics.Metrics
	monitors    repository.MonitorRepository
	incidents   *service.IncidentService
	checks      *service.MonitorCheckService
	stats       *service.MonitorStatsService
}

func NewServer(
	cfg config.Config,
	monitors repository.MonitorRepository,
	checkResults repository.CheckResultRepository,
	incidents repository.IncidentRepository,
	publisher queue.Publisher,
	statsCache cache.MonitorStatsCache,
) *Server {
	incidentSvc := service.NewIncidentService(checkResults, incidents, repository.NewMemoryTransactor())
	incidentSvc.SetStatsCache(statsCache)
	checks := service.NewMonitorCheckService(monitors, checkResults, nil, publisher, incidentSvc)
	checks.SetStatsCache(statsCache)
	return &Server{
		addr:        cfg.Addr,
		corsOrigins: cfg.CORSOrigins,
		metrics:     metrics.Default(),
		monitors:    monitors,
		incidents:   incidentSvc,
		checks:      checks,
		stats:       service.NewMonitorStatsService(monitors, checkResults, incidents, statsCache),
	}
}

func (s *Server) Handler() http.Handler {
	mux := s.routes()
	mux.Handle("GET /metrics", metrics.MetricsHandler(s.metrics))
	return corsMiddleware(s.corsOrigins, metrics.HTTPMiddleware(s.metrics, mux))
}

func (s *Server) Start() error {
	httpServer := &http.Server{
		Addr:    s.addr,
		Handler: s.Handler(),
	}

	return httpServer.ListenAndServe()
}
