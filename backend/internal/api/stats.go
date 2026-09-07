package api

import (
	"net/http"
)

func (s *Server) getMonitorStats(w http.ResponseWriter, r *http.Request) {
	id, ok := parseMonitorID(w, r)
	if !ok {
		return
	}

	stats, err := s.stats.GetStats(r.Context(), id)
	if !writeMonitorError(w, err) {
		return
	}

	writeJSON(w, http.StatusOK, stats)
}
