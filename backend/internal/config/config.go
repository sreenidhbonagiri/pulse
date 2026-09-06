package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds app settings loaded from environment variables.
type Config struct {
	Addr        string
	DatabaseURL string
}

func Load() Config {
	// Load .env if present. Missing files are ignored so Docker/prod still work.
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	return Config{
		Addr:        env("API_ADDR", ":8080"),
		DatabaseURL: env("DATABASE_URL", "postgres://pulse:pulse@localhost:5432/pulse?sslmode=disable"),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
