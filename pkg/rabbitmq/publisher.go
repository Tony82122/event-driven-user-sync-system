package rabbitmq

import (
	"context"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const ExchangeName = "events"

// Publisher publishes messages to the RabbitMQ exchange.
type Publisher struct {
	channel *amqp.Channel
}

// NewPublisher creates a new publisher and declares the topic exchange.
func NewPublisher(conn *Connection) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Declare topic exchange
	err = ch.ExchangeDeclare(
		ExchangeName,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &Publisher{channel: ch}, nil
}

// Publish sends a message to the exchange with the given routing key.
func (p *Publisher) Publish(routingKey string, body []byte, correlationID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Printf("[Publisher] Publishing event: routing_key=%s correlation_id=%s", routingKey, correlationID)

	return p.channel.PublishWithContext(
		ctx,
		ExchangeName,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: correlationID,
			Body:          body,
			DeliveryMode:  amqp.Persistent,
			Timestamp:     time.Now(),
		},
	)
}

// Close closes the publisher channel.
func (p *Publisher) Close() error {
	if p.channel != nil {
		return p.channel.Close()
	}
	return nil
}
