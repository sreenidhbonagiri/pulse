package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

// Redis stores monitor stats in Redis. Command errors are returned to the
// caller so the service can fall back to PostgreSQL.
type Redis struct {
	client *redis.Client
}

func Dial(ctx context.Context, url string) (*Redis, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("redis ping failed, stats will be served from postgres: %v", err)
	}

	return &Redis{client: client}, nil
}

func (r *Redis) GetMonitorStats(ctx context.Context, monitorID uuid.UUID) (*models.MonitorStats, error) {
	raw, err := r.client.Get(ctx, statsKey(monitorID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var stats models.MonitorStats
	if err := json.Unmarshal(raw, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *Redis) SetMonitorStats(ctx context.Context, monitorID uuid.UUID, stats models.MonitorStats, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	raw, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, statsKey(monitorID), raw, ttl).Err()
}

func (r *Redis) DeleteMonitorStats(ctx context.Context, monitorID uuid.UUID) error {
	return r.client.Del(ctx, statsKey(monitorID)).Err()
}

func (r *Redis) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}
