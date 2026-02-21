package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"awesomeProject/internal/analytics"
	"awesomeProject/pkg/config"
	"awesomeProject/pkg/postgres"
	"awesomeProject/pkg/rabbitmq"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[Analytics] Starting analytics-consumer...")

	cfg := config.Load()

	// Connect to PostgreSQL
	db, err := postgres.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[Analytics] Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := postgres.RunMigrations(db, "analytics"); err != nil {
		log.Fatalf("[Analytics] Failed to run migrations: %v", err)
	}

	// Connect to RabbitMQ
	rmqConn, err := rabbitmq.Connect(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("[Analytics] Failed to connect to RabbitMQ: %v", err)
	}
	defer rmqConn.Close()

	// Create consumer
	consumer := analytics.NewConsumer(db)

	consumerCfg := rabbitmq.ConsumerConfig{
		QueueName:    "analytics.user.events",
		DLQName:      "dlq.analytics.user.events",
		RoutingKeys:  []string{"user.created", "user.updated", "user.deleted"},
		ConsumerName: "analytics-consumer",
	}

	if err := rabbitmq.SetupConsumer(rmqConn, consumerCfg, consumer.HandleMessage); err != nil {
		log.Fatalf("[Analytics] Failed to setup consumer: %v", err)
	}

	log.Println("[Analytics] Consumer is running. Waiting for messages...")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[Analytics] Shutting down...")
}
