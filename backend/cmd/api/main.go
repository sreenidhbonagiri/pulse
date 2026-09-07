package main

import (
	"context"
	"log"

	"github.com/sreenidhbonagiri/pulse/backend/internal/api"
	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
	"github.com/sreenidhbonagiri/pulse/backend/internal/database"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool); err != nil {
		log.Fatalf("database migrations: %v", err)
	}

	monitorRepo := repository.NewPostgresMonitorRepository(pool)
	checkResultRepo := repository.NewPostgresCheckResultRepository(pool)
	server := api.NewServer(cfg, monitorRepo, checkResultRepo)

	log.Printf("starting Pulse API on %s", cfg.Addr)
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
