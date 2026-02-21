package rabbitmq

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ConsumerConfig holds configuration for setting up a consumer.
type ConsumerConfig struct {
	QueueName    string
	DLQName      string
	RoutingKeys  []string
	ConsumerName string
}

// MessageHandler is a function that processes a delivered message.
// Return nil to ack, return error to nack (triggers retry/DLQ).
type MessageHandler func(delivery amqp.Delivery) error

// SetupConsumer declares queues (main + DLQ), binds them, and starts consuming.
func SetupConsumer(conn *Connection, cfg ConsumerConfig, handler MessageHandler) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}

	// Declare the topic exchange (idempotent)
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
		return err
	}

	// Declare DLQ
	_, err = ch.QueueDeclare(
		cfg.DLQName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	// Declare main queue with DLQ settings
	args := amqp.Table{
		"x-dead-letter-exchange":    "",          // default exchange
		"x-dead-letter-routing-key": cfg.DLQName, // route to DLQ
	}

	_, err = ch.QueueDeclare(
		cfg.QueueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		args,
	)
	if err != nil {
		return err
	}

	// Bind queue to exchange with routing keys
	for _, key := range cfg.RoutingKeys {
		err = ch.QueueBind(
			cfg.QueueName,
			key,
			ExchangeName,
			false,
			nil,
		)
		if err != nil {
			return err
		}
	}

	// Set prefetch count
	err = ch.Qos(1, 0, false)
	if err != nil {
		return err
	}

	// Start consuming
	msgs, err := ch.Consume(
		cfg.QueueName,
		cfg.ConsumerName,
		false, // auto-ack = false (manual ack)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			log.Printf("[%s] Received message: routing_key=%s correlation_id=%s",
				cfg.ConsumerName, msg.RoutingKey, msg.CorrelationId)

			if err := handler(msg); err != nil {
				log.Printf("[%s] Error processing message: %v — nacking (will go to DLQ)",
					cfg.ConsumerName, err)
				_ = msg.Nack(false, false) // don't requeue — goes to DLQ
			} else {
				_ = msg.Ack(false)
			}
		}
	}()

	log.Printf("[%s] Consumer started, listening on queue: %s", cfg.ConsumerName, cfg.QueueName)
	return nil
}
