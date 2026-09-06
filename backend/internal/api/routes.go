package api

import "net/http"

func routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	return mux
}
