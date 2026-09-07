package api

import (
	"net/http"

	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
	"github.com/sreenidhbonagiri/pulse/backend/internal/service"
)

type Server struct {
	addr     string
	monitors repository.MonitorRepository
	checks   *service.MonitorCheckService
}

func NewServer(
	cfg config.Config,
	monitors repository.MonitorRepository,
	checkResults repository.CheckResultRepository,
	publisher queue.Publisher,
) *Server {
	return &Server{
		addr:     cfg.Addr,
		monitors: monitors,
		checks:   service.NewMonitorCheckService(monitors, checkResults, nil, publisher),
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
