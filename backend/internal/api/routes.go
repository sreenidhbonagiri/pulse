package api

import "net/http"

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", health)

	mux.HandleFunc("POST /api/monitors", s.createMonitor)
	mux.HandleFunc("GET /api/monitors", s.listMonitors)
	mux.HandleFunc("GET /api/monitors/{id}", s.getMonitor)
	mux.HandleFunc("PUT /api/monitors/{id}", s.updateMonitor)
	mux.HandleFunc("DELETE /api/monitors/{id}", s.deleteMonitor)
	mux.HandleFunc("POST /api/monitors/{id}/check", s.checkMonitor)
	mux.HandleFunc("GET /api/monitors/{id}/checks", s.listMonitorChecks)
	mux.HandleFunc("GET /api/monitors/{id}/incidents/active", s.getActiveIncident)
	mux.HandleFunc("GET /api/monitors/{id}/incidents", s.listMonitorIncidents)

	return mux
}
