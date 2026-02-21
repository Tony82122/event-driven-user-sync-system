package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Clear any env vars that might be set
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("RABBITMQ_URL")
	os.Unsetenv("API_PORT")

	cfg := Load()

	if cfg.DatabaseURL != "postgres://postgres:postgres@postgres:5432/appdb?sslmode=disable" {
		t.Errorf("unexpected DatabaseURL: %s", cfg.DatabaseURL)
	}
	if cfg.RabbitMQURL != "amqp://guest:guest@rabbitmq:5672/" {
		t.Errorf("unexpected RabbitMQURL: %s", cfg.RabbitMQURL)
	}
	if cfg.APIPort != "8080" {
		t.Errorf("unexpected APIPort: %s", cfg.APIPort)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://custom:pass@host:5432/db")
	os.Setenv("RABBITMQ_URL", "amqp://user:pass@rmq:5672/")
	os.Setenv("API_PORT", "9090")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("RABBITMQ_URL")
		os.Unsetenv("API_PORT")
	}()

	cfg := Load()

	if cfg.DatabaseURL != "postgres://custom:pass@host:5432/db" {
		t.Errorf("unexpected DatabaseURL: %s", cfg.DatabaseURL)
	}
	if cfg.RabbitMQURL != "amqp://user:pass@rmq:5672/" {
		t.Errorf("unexpected RabbitMQURL: %s", cfg.RabbitMQURL)
	}
	if cfg.APIPort != "9090" {
		t.Errorf("unexpected APIPort: %s", cfg.APIPort)
	}
}

func TestLoadForService(t *testing.T) {
	os.Unsetenv("DATABASE_URL")
	os.Setenv("CRM_DATABASE_URL", "postgres://crm@host:5432/crm_db")
	defer os.Unsetenv("CRM_DATABASE_URL")

	cfg := LoadForService("CRM")

	if cfg.DatabaseURL != "postgres://crm@host:5432/crm_db" {
		t.Errorf("unexpected DatabaseURL: %s", cfg.DatabaseURL)
	}
}

func TestGetEnvFallback(t *testing.T) {
	os.Unsetenv("NONEXISTENT_KEY")
	val := getEnv("NONEXISTENT_KEY", "fallback-value")
	if val != "fallback-value" {
		t.Errorf("expected fallback-value, got %s", val)
	}
}
