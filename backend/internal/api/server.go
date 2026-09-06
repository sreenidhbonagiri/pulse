package api

import (
	"net/http"

	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
)

func Start(cfg config.Config) error {
	server := &http.Server{
		Addr:    cfg.Addr,
		Handler: routes(),
	}

	return server.ListenAndServe()
}
