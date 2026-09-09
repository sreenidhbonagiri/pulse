package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
	"github.com/sreenidhbonagiri/pulse/backend/internal/database"
	"github.com/sreenidhbonagiri/pulse/backend/internal/metrics"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
	"github.com/sreenidhbonagiri/pulse/backend/internal/scheduler"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	m := metrics.Init("scheduler")
	metrics.StartServer(ctx, cfg.SchedulerMetricsAddr, m)

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool); err != nil {
		log.Fatalf("database migrations: %v", err)
	}

	q, err := queue.New(ctx, queue.Config{
		Provider:    cfg.QueueProvider,
		RabbitMQURL: cfg.RabbitMQURL,
		SQSQueueURL: cfg.SQSQueueURL,
		SQSDLQURL:   cfg.SQSDLQURL,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer q.Close()

	log.Printf("Pulse scheduler polling every %s", scheduler.DefaultPollInterval)
	if err := scheduler.New(repository.NewPostgresMonitorRepository(pool), q).Run(ctx); err != nil {
		log.Fatal(err)
	}
	log.Printf("Pulse scheduler stopped")
}
