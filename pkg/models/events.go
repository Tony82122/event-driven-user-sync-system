package models

import "time"

// EventType represents the type of domain event.
type EventType string

const (
	EventUserCreated EventType = "user.created"
	EventUserUpdated EventType = "user.updated"
	EventUserDeleted EventType = "user.deleted"
)

// UserEvent represents an event related to a user.
type UserEvent struct {
	EventID       string    `json:"event_id"`
	CorrelationID string    `json:"correlation_id"`
	EventType     EventType `json:"event_type"`
	Timestamp     time.Time `json:"timestamp"`
	Data          User      `json:"data"`
}
