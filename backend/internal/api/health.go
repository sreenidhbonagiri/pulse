package api

import "net/http"

func health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:  "ok",
		Service: "pulse-api",
	})
}

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}
