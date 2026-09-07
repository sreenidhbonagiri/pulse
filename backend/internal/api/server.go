package api

import (
	"net/http"

	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
	"github.com/sreenidhbonagiri/pulse/backend/internal/monitoring"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
	"github.com/sreenidhbonagiri/pulse/backend/internal/service"
)

type Server struct {
	addr     string
	monitors repository.MonitorRepository
	checks   *service.MonitorCheckService
}

func NewServer(cfg config.Config, monitors repository.MonitorRepository, checkResults repository.CheckResultRepository) *Server {
	return NewServerWithChecker(cfg, monitors, checkResults, monitoring.NewChecker(nil))
}

func NewServerWithChecker(
	cfg config.Config,
	monitors repository.MonitorRepository,
	checkResults repository.CheckResultRepository,
	checker service.HTTPChecker,
) *Server {
	return &Server{
		addr:     cfg.Addr,
		monitors: monitors,
		checks:   service.NewMonitorCheckService(monitors, checkResults, checker),
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
