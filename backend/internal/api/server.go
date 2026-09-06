package api

import (
	"fmt"
	"net/http"

	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
)

func Start(cfg config.Config) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Pulse API is running")
	})

	return http.ListenAndServe(cfg.Addr, mux)
}
