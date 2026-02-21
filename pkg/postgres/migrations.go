package postgres

import (
	"database/sql"
	"log"
)

// RunMigrations executes database migrations.
func RunMigrations(db *sql.DB, service string) error {
	migrations := getServiceMigrations(service)
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return err
		}
	}
	log.Printf("Migrations completed for service: %s", service)
	return nil
}

func getServiceMigrations(service string) []string {
	common := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(36) PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS idempotency_keys (
			event_id VARCHAR(36) PRIMARY KEY,
			processed_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
	}

	switch service {
	case "api":
		return common
	case "crm":
		return []string{
			`CREATE TABLE IF NOT EXISTS idempotency_keys (
				event_id VARCHAR(36) PRIMARY KEY,
				processed_at TIMESTAMP NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS crm_sync_log (
				id SERIAL PRIMARY KEY,
				event_id VARCHAR(36) NOT NULL,
				correlation_id VARCHAR(36),
				event_type VARCHAR(50) NOT NULL,
				user_id VARCHAR(36) NOT NULL,
				user_email VARCHAR(255),
				user_name VARCHAR(255),
				synced_at TIMESTAMP NOT NULL DEFAULT NOW()
			)`,
		}
	case "analytics":
		return []string{
			`CREATE TABLE IF NOT EXISTS idempotency_keys (
				event_id VARCHAR(36) PRIMARY KEY,
				processed_at TIMESTAMP NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS analytics_metrics (
				id SERIAL PRIMARY KEY,
				metric_date DATE NOT NULL,
				event_type VARCHAR(50) NOT NULL,
				count INTEGER NOT NULL DEFAULT 0,
				UNIQUE(metric_date, event_type)
			)`,
		}
	default:
		return common
	}
}
