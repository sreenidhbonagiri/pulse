package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds app settings loaded from environment variables.
type Config struct {
	Addr        string
	DatabaseURL string
	RabbitMQURL string
	RedisURL    string
	CORSOrigins string
}

func Load() Config {
	// Load .env if present. Missing files are ignored so Docker/prod still work.
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	return Config{
		Addr:        env("API_ADDR", ":8080"),
		DatabaseURL: env("DATABASE_URL", "postgres://pulse:pulse@localhost:5432/pulse?sslmode=disable"),
		RabbitMQURL: env("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RedisURL:    env("REDIS_URL", "redis://localhost:6379/0"),
		CORSOrigins: env("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
