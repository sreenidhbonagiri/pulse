package api

import (
	"net/http"

	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

type Server struct {
	addr         string
	monitors     repository.MonitorRepository
	checkResults repository.CheckResultRepository
}

func NewServer(cfg config.Config, monitors repository.MonitorRepository, checkResults repository.CheckResultRepository) *Server {
	return &Server{
		addr:         cfg.Addr,
		monitors:     monitors,
		checkResults: checkResults,
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
