package api

import (
	"errors"
	"net/http"

	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

func (s *Server) listMonitorIncidents(w http.ResponseWriter, r *http.Request) {
	id, ok := parseMonitorID(w, r)
	if !ok {
		return
	}

	if _, err := s.monitors.GetByID(r.Context(), id); !writeMonitorError(w, err) {
		return
	}

	incidents, err := s.incidents.ListByMonitorID(r.Context(), id, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list incidents")
		return
	}

	writeJSON(w, http.StatusOK, incidents)
}

func (s *Server) getActiveIncident(w http.ResponseWriter, r *http.Request) {
	id, ok := parseMonitorID(w, r)
	if !ok {
		return
	}

	if _, err := s.monitors.GetByID(r.Context(), id); !writeMonitorError(w, err) {
		return
	}

	incident, err := s.incidents.GetOpenByMonitorID(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "no open incident")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get incident")
		return
	}

	writeJSON(w, http.StatusOK, incident)
}
