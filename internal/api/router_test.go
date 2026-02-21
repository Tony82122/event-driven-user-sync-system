package api

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestNewRouter_RoutesExist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	pub := &mockPublisher{}
	handler := NewUserHandler(db, pub)
	router := NewRouter(handler)

	routes := router.Routes()
	expectedRoutes := map[string]string{
		"GET /health":    "health",
		"POST /users":    "create",
		"PUT /users/:id": "update",
		"GET /users/:id": "get",
		"GET /users":     "list",
	}

	found := make(map[string]bool)
	for _, r := range routes {
		key := r.Method + " " + r.Path
		if _, ok := expectedRoutes[key]; ok {
			found[key] = true
		}
	}

	for key, desc := range expectedRoutes {
		if !found[key] {
			t.Errorf("missing route %s (%s)", key, desc)
		}
	}
}

func TestSwaggerRouteRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	pub := &mockPublisher{}
	handler := NewUserHandler(db, pub)
	router := NewRouter(handler)

	// Verify the swagger route is registered
	routes := router.Routes()
	found := false
	for _, r := range routes {
		if r.Method == "GET" && r.Path == "/swagger/*any" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected /swagger/*any route to be registered")
	}
}
