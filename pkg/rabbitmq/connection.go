package rabbitmq

import (
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Connection wraps an AMQP connection with reconnect logic.
type Connection struct {
	URL  string
	Conn *amqp.Connection
}

// Connect establishes a connection to RabbitMQ with retries.
func Connect(url string) (*Connection, error) {
	var conn *amqp.Connection
	var err error

	for i := 0; i < 30; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			log.Println("Connected to RabbitMQ")
			return &Connection{URL: url, Conn: conn}, nil
		}
		log.Printf("Failed to connect to RabbitMQ: %v, retrying in 2s...", err)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("could not connect to RabbitMQ after 30 attempts: %w", err)
}

// Channel opens a new AMQP channel.
func (c *Connection) Channel() (*amqp.Channel, error) {
	return c.Conn.Channel()
}

// Close closes the connection.
func (c *Connection) Close() error {
	if c.Conn != nil {
		return c.Conn.Close()
	}
	return nil
}
