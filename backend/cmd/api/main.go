package main

import (
	"context"
	"log"

	"github.com/sreenidhbonagiri/pulse/backend/internal/api"
	"github.com/sreenidhbonagiri/pulse/backend/internal/cache"
	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
	"github.com/sreenidhbonagiri/pulse/backend/internal/database"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
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

	rmq, err := queue.Dial(cfg.RabbitMQURL)
	if err != nil {
		log.Fatal(err)
	}
	defer rmq.Close()

	statsCache, err := cache.Dial(ctx, cfg.RedisURL)
	if err != nil {
		log.Printf("redis disabled, stats will be served from postgres: %v", err)
	}
	if statsCache != nil {
		defer statsCache.Close()
	}

	monitorRepo := repository.NewPostgresMonitorRepository(pool)
	checkResultRepo := repository.NewPostgresCheckResultRepository(pool)
	incidentRepo := repository.NewPostgresIncidentRepository(pool)
	server := api.NewServer(cfg, monitorRepo, checkResultRepo, incidentRepo, rmq, statsCache)

	log.Printf("starting Pulse API on %s", cfg.Addr)
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
