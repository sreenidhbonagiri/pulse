package api

import (
	"errors"
	"net/http"

	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

func (s *Server) checkMonitor(w http.ResponseWriter, r *http.Request) {
	id, ok := parseMonitorID(w, r)
	if !ok {
		return
	}

	job, err := s.checks.EnqueueCheck(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "monitor not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enqueue check")
		return
	}

	writeJSON(w, http.StatusAccepted, job)
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
