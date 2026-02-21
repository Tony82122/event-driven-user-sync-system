package postgres

import (
	"testing"
)

func TestGetServiceMigrations_API(t *testing.T) {
	migrations := getServiceMigrations("api")
	if len(migrations) != 2 {
		t.Fatalf("expected 2 migrations for api, got %d", len(migrations))
	}
}

func TestGetServiceMigrations_CRM(t *testing.T) {
	migrations := getServiceMigrations("crm")
	if len(migrations) != 2 {
		t.Fatalf("expected 2 migrations for crm, got %d", len(migrations))
	}
}

func TestGetServiceMigrations_Analytics(t *testing.T) {
	migrations := getServiceMigrations("analytics")
	if len(migrations) != 2 {
		t.Fatalf("expected 2 migrations for analytics, got %d", len(migrations))
	}
}

func TestGetServiceMigrations_Default(t *testing.T) {
	migrations := getServiceMigrations("unknown")
	if len(migrations) != 2 {
		t.Fatalf("expected 2 migrations for unknown (default), got %d", len(migrations))
	}
}
