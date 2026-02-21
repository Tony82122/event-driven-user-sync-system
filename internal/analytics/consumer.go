package analytics

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"

	"awesomeProject/pkg/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer handles analytics events.
type Consumer struct {
	DB               *sql.DB
	SimulateFailures bool
}

// NewConsumer creates a new analytics consumer.
func NewConsumer(db *sql.DB) *Consumer {
	return &Consumer{DB: db, SimulateFailures: true}
}

// HandleMessage processes a user event for analytics.
func (c *Consumer) HandleMessage(delivery amqp.Delivery) error {
	var event models.UserEvent
	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		log.Printf("[Analytics] Failed to unmarshal event: %v correlation_id=%s", err, delivery.CorrelationId)
		return err
	}

	log.Printf("[Analytics] Processing event: type=%s event_id=%s correlation_id=%s user_id=%s",
		event.EventType, event.EventID, event.CorrelationID, event.Data.ID)

	// Idempotency check
	var exists bool
	err := c.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM idempotency_keys WHERE event_id = $1)", event.EventID).Scan(&exists)
	if err != nil {
		log.Printf("[Analytics] Error checking idempotency: %v correlation_id=%s", err, event.CorrelationID)
		return err
	}
	if exists {
		log.Printf("[Analytics] Duplicate event ignored: event_id=%s correlation_id=%s", event.EventID, event.CorrelationID)
		return nil
	}

	// Simulate random failure (10% chance)
	if c.SimulateFailures && rand.Intn(10) == 0 {
		log.Printf("[Analytics] Simulated failure! event_id=%s correlation_id=%s", event.EventID, event.CorrelationID)
		return fmt.Errorf("simulated analytics failure")
	}

	// Aggregate metrics — upsert count by date and event type
	metricDate := event.Timestamp.Format("2006-01-02")
	_, err = c.DB.Exec(
		`INSERT INTO analytics_metrics (metric_date, event_type, count)
		 VALUES ($1, $2, 1)
		 ON CONFLICT (metric_date, event_type)
		 DO UPDATE SET count = analytics_metrics.count + 1`,
		metricDate, string(event.EventType),
	)
	if err != nil {
		log.Printf("[Analytics] Error upserting metrics: %v correlation_id=%s", err, event.CorrelationID)
		return err
	}

	// Record idempotency key
	_, _ = c.DB.Exec("INSERT INTO idempotency_keys (event_id) VALUES ($1) ON CONFLICT DO NOTHING", event.EventID)

	log.Printf("[Analytics] Metrics updated: date=%s type=%s correlation_id=%s",
		metricDate, event.EventType, event.CorrelationID)

	return nil
}
