package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"awesomeProject/internal/crm"
	"awesomeProject/pkg/config"
	"awesomeProject/pkg/postgres"
	"awesomeProject/pkg/rabbitmq"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[CRM] Starting crm-consumer...")

	cfg := config.Load()

	// Connect to PostgreSQL
	db, err := postgres.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[CRM] Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := postgres.RunMigrations(db, "crm"); err != nil {
		log.Fatalf("[CRM] Failed to run migrations: %v", err)
	}

	// Connect to RabbitMQ
	rmqConn, err := rabbitmq.Connect(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("[CRM] Failed to connect to RabbitMQ: %v", err)
	}
	defer rmqConn.Close()

	// Create consumer
	consumer := crm.NewConsumer(db)

	consumerCfg := rabbitmq.ConsumerConfig{
		QueueName:    "crm.user.events",
		DLQName:      "dlq.crm.user.events",
		RoutingKeys:  []string{"user.created", "user.updated", "user.deleted"},
		ConsumerName: "crm-consumer",
	}

	if err := rabbitmq.SetupConsumer(rmqConn, consumerCfg, consumer.HandleMessage); err != nil {
		log.Fatalf("[CRM] Failed to setup consumer: %v", err)
	}

	log.Println("[CRM] Consumer is running. Waiting for messages...")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[CRM] Shutting down...")
}
