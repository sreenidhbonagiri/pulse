package api

import (
	"encoding/json"
	"net/http"
)

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := healthResponse{
		Status:  "ok",
		Service: "pulse-api",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}
