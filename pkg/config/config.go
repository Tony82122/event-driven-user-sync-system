package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the application.
type Config struct {
	// PostgreSQL
	DatabaseURL string

	// RabbitMQ
	RabbitMQURL string

	// API
	APIPort string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/appdb?sslmode=disable"),
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"),
		APIPort:     getEnv("API_PORT", "8080"),
	}
}

// LoadForService returns config with a service-specific DATABASE_URL env var fallback.
func LoadForService(service string) *Config {
	cfg := Load()
	envKey := fmt.Sprintf("%s_DATABASE_URL", service)
	if v := os.Getenv(envKey); v != "" {
		cfg.DatabaseURL = v
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
