package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"awesomeProject/internal/api"
	"awesomeProject/pkg/config"
	"awesomeProject/pkg/postgres"
	"awesomeProject/pkg/rabbitmq"

	_ "awesomeProject/docs"
)

// @title           Event-Driven User API
// @version         1.0
// @description     A RESTful API that publishes user events to RabbitMQ for async processing by CRM and Analytics consumers.
// @host            localhost:8080
// @BasePath        /
// @schemes         http
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[API] Starting api-service...")

	cfg := config.Load()

	// Connect to PostgreSQL
	db, err := postgres.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[API] Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := postgres.RunMigrations(db, "api"); err != nil {
		log.Fatalf("[API] Failed to run migrations: %v", err)
	}

	// Connect to RabbitMQ
	rmqConn, err := rabbitmq.Connect(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("[API] Failed to connect to RabbitMQ: %v", err)
	}
	defer rmqConn.Close()

	// Create publisher
	publisher, err := rabbitmq.NewPublisher(rmqConn)
	if err != nil {
		log.Fatalf("[API] Failed to create publisher: %v", err)
	}
	defer publisher.Close()

	// Setup handlers and router
	handler := api.NewUserHandler(db, publisher)
	router := api.NewRouter(handler)

	// HTTP server with graceful shutdown
	srv := &http.Server{
		Addr:    ":" + cfg.APIPort,
		Handler: router,
	}

	go func() {
		log.Printf("[API] Listening on port %s", cfg.APIPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[API] Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[API] Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[API] Server forced to shutdown: %v", err)
	}
	log.Println("[API] Server exited gracefully")
}
