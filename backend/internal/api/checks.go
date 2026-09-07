package api

import "net/http"

func (s *Server) checkMonitor(w http.ResponseWriter, r *http.Request) {
	id, ok := parseMonitorID(w, r)
	if !ok {
		return
	}

	result, err := s.checks.RunCheck(r.Context(), id)
	if !writeMonitorError(w, err) {
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) listMonitorChecks(w http.ResponseWriter, r *http.Request) {
	id, ok := parseMonitorID(w, r)
	if !ok {
		return
	}

	results, err := s.checks.ListChecks(r.Context(), id, 0)
	if !writeMonitorError(w, err) {
		return
	}

	writeJSON(w, http.StatusOK, results)
}
