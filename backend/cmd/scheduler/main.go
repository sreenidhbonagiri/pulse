package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
	"github.com/sreenidhbonagiri/pulse/backend/internal/database"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
	"github.com/sreenidhbonagiri/pulse/backend/internal/scheduler"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

	log.Printf("Pulse scheduler polling every %s", scheduler.DefaultPollInterval)
	if err := scheduler.New(repository.NewPostgresMonitorRepository(pool), rmq).Run(ctx); err != nil {
		log.Fatal(err)
	}
	log.Printf("Pulse scheduler stopped")
}
