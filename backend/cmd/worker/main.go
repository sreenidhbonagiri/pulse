package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sreenidhbonagiri/pulse/backend/internal/cache"
	"github.com/sreenidhbonagiri/pulse/backend/internal/config"
	"github.com/sreenidhbonagiri/pulse/backend/internal/database"
	"github.com/sreenidhbonagiri/pulse/backend/internal/metrics"
	"github.com/sreenidhbonagiri/pulse/backend/internal/monitoring"
	"github.com/sreenidhbonagiri/pulse/backend/internal/queue"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
	"github.com/sreenidhbonagiri/pulse/backend/internal/service"
	"github.com/sreenidhbonagiri/pulse/backend/internal/worker"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	m := metrics.Init("worker")
	metrics.StartServer(ctx, cfg.WorkerMetricsAddr, m)

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

	checkResults := repository.NewPostgresCheckResultRepository(pool)
	incidentRepo := repository.NewPostgresIncidentRepository(pool)
	incidents := service.NewIncidentService(
		checkResults,
		incidentRepo,
		repository.NewPostgresTransactor(pool),
	)
	checks := service.NewMonitorCheckService(
		repository.NewPostgresMonitorRepository(pool),
		checkResults,
		monitoring.NewChecker(nil),
		nil,
		incidents,
	)

	if count, err := incidentRepo.CountOpen(ctx); err != nil {
		log.Printf("could not load active incident count: %v", err)
	} else {
		m.SetActiveIncidents(float64(count))
	}

	statsCache, err := cache.Dial(ctx, cfg.RedisURL)
	if err != nil {
		log.Printf("redis disabled, stats cache invalidation skipped: %v", err)
	}
	if statsCache != nil {
		defer statsCache.Close()
		incidents.SetStatsCache(statsCache)
		checks.SetStatsCache(statsCache)
	}

	log.Printf("Pulse worker listening on queue %s", queue.MonitorChecksQueue)
	if err := rmq.Consume(ctx, worker.New(checks).HandleJob); err != nil {
		log.Fatal(err)
	}
	log.Printf("Pulse worker stopped")
}
