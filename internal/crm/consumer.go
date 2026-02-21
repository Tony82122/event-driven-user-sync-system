package crm

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"

	"awesomeProject/pkg/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer handles CRM sync events.
type Consumer struct {
	DB *sql.DB
}

// NewConsumer creates a new CRM consumer.
func NewConsumer(db *sql.DB) *Consumer {
	return &Consumer{DB: db}
}

// HandleMessage processes a user event for CRM sync.
func (c *Consumer) HandleMessage(delivery amqp.Delivery) error {
	var event models.UserEvent
	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		log.Printf("[CRM] Failed to unmarshal event: %v correlation_id=%s", err, delivery.CorrelationId)
		return err
	}

	log.Printf("[CRM] Processing event: type=%s event_id=%s correlation_id=%s user_id=%s",
		event.EventType, event.EventID, event.CorrelationID, event.Data.ID)

	// Idempotency check
	var exists bool
	err := c.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM idempotency_keys WHERE event_id = $1)", event.EventID).Scan(&exists)
	if err != nil {
		log.Printf("[CRM] Error checking idempotency: %v correlation_id=%s", err, event.CorrelationID)
		return err
	}
	if exists {
		log.Printf("[CRM] Duplicate event ignored: event_id=%s correlation_id=%s", event.EventID, event.CorrelationID)
		return nil // Already processed — ack it
	}

	// Simulate random failure (10% chance) to demonstrate retry + DLQ
	if rand.Intn(10) == 0 {
		log.Printf("[CRM] Simulated failure! event_id=%s correlation_id=%s", event.EventID, event.CorrelationID)
		return fmt.Errorf("simulated CRM sync failure")
	}

	// Simulate CRM sync — write to crm_sync_log
	_, err = c.DB.Exec(
		`INSERT INTO crm_sync_log (event_id, correlation_id, event_type, user_id, user_email, user_name)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		event.EventID, event.CorrelationID, string(event.EventType),
		event.Data.ID, event.Data.Email, event.Data.Name,
	)
	if err != nil {
		log.Printf("[CRM] Error writing sync log: %v correlation_id=%s", err, event.CorrelationID)
		return err
	}

	// Record idempotency key
	_, _ = c.DB.Exec("INSERT INTO idempotency_keys (event_id) VALUES ($1) ON CONFLICT DO NOTHING", event.EventID)

	log.Printf("[CRM] Successfully synced: event_id=%s type=%s user=%s correlation_id=%s",
		event.EventID, event.EventType, event.Data.Email, event.CorrelationID)

	return nil
}
