package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

func (s *Server) createMonitor(w http.ResponseWriter, r *http.Request) {
	req, err := decodeMonitorRequest(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	monitor := req.toMonitor()
	if err := s.monitors.Create(r.Context(), monitor); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create monitor")
		return
	}

	w.Header().Set("Location", "/api/monitors/"+monitor.ID.String())
	writeJSON(w, http.StatusCreated, monitor)
}

func (s *Server) listMonitors(w http.ResponseWriter, r *http.Request) {
	monitors, err := s.monitors.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list monitors")
		return
	}

	writeJSON(w, http.StatusOK, monitors)
}

func (s *Server) getMonitor(w http.ResponseWriter, r *http.Request) {
	id, ok := parseMonitorID(w, r)
	if !ok {
		return
	}

	monitor, err := s.monitors.GetByID(r.Context(), id)
	if !writeMonitorError(w, err) {
		return
	}

	writeJSON(w, http.StatusOK, monitor)
}

func (s *Server) updateMonitor(w http.ResponseWriter, r *http.Request) {
	id, ok := parseMonitorID(w, r)
	if !ok {
		return
	}

	req, err := decodeMonitorRequest(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	existing, err := s.monitors.GetByID(r.Context(), id)
	if !writeMonitorError(w, err) {
		return
	}

	monitor := applyMonitorUpdate(existing, req)
	if err := s.monitors.Update(r.Context(), monitor); !writeMonitorError(w, err) {
		return
	}

	writeJSON(w, http.StatusOK, monitor)
}

func (s *Server) deleteMonitor(w http.ResponseWriter, r *http.Request) {
	id, ok := parseMonitorID(w, r)
	if !ok {
		return
	}

	if err := s.monitors.Delete(r.Context(), id); !writeMonitorError(w, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseMonitorID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid monitor id")
		return uuid.Nil, false
	}
	return id, true
}

func writeMonitorError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "monitor not found")
		return false
	}
	writeError(w, http.StatusInternalServerError, "database error")
	return false
}

func applyMonitorUpdate(existing *models.Monitor, req monitorRequest) *models.Monitor {
	updated := req.toMonitor()
	updated.ID = existing.ID
	updated.UserID = existing.UserID
	if req.IsActive == nil {
		updated.IsActive = existing.IsActive
	}
	return updated
}
