package main

import (
	"log"

	"github.com/sreenidhbonagiri/pulse/backend/internal/api"
	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
)

func main() {
	cfg := config.Load()
	log.Printf("starting Pulse API on %s", cfg.Addr)

	if err := api.Start(cfg); err != nil {
		log.Fatal(err)
	}
}
