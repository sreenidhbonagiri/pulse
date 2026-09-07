package api

import (
	"net/http"

	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
	"github.com/sreenidhbonagiri/pulse/backend/internal/service"
)

type Server struct {
	addr      string
	monitors  repository.MonitorRepository
	incidents *service.IncidentService
	checks    *service.MonitorCheckService
}

func NewServer(
	cfg config.Config,
	monitors repository.MonitorRepository,
	checkResults repository.CheckResultRepository,
	incidents repository.IncidentRepository,
	publisher queue.Publisher,
) *Server {
	incidentSvc := service.NewIncidentService(checkResults, incidents, repository.NewMemoryTransactor())
	return &Server{
		addr:      cfg.Addr,
		monitors:  monitors,
		incidents: incidentSvc,
		checks:    service.NewMonitorCheckService(monitors, checkResults, nil, publisher, incidentSvc),
	}
}

func (s *Server) Handler() http.Handler {
	return s.routes()
}

func (s *Server) Start() error {
	httpServer := &http.Server{
		Addr:    s.addr,
		Handler: s.Handler(),
	}

	return httpServer.ListenAndServe()
}
