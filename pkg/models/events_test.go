package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		et       EventType
		expected string
	}{
		{"user created", EventUserCreated, "user.created"},
		{"user updated", EventUserUpdated, "user.updated"},
		{"user deleted", EventUserDeleted, "user.deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.et) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.et))
			}
		})
	}
}

func TestUserEventJSON(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	event := UserEvent{
		EventID:       "evt-123",
		CorrelationID: "corr-456",
		EventType:     EventUserCreated,
		Timestamp:     now,
		Data: User{
			ID:        "user-789",
			Email:     "test@example.com",
			Name:      "Test User",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal UserEvent: %v", err)
	}

	var decoded UserEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal UserEvent: %v", err)
	}

	if decoded.EventID != event.EventID {
		t.Errorf("EventID: expected %q, got %q", event.EventID, decoded.EventID)
	}
	if decoded.CorrelationID != event.CorrelationID {
		t.Errorf("CorrelationID: expected %q, got %q", event.CorrelationID, decoded.CorrelationID)
	}
	if decoded.EventType != event.EventType {
		t.Errorf("EventType: expected %q, got %q", event.EventType, decoded.EventType)
	}
	if decoded.Data.ID != event.Data.ID {
		t.Errorf("Data.ID: expected %q, got %q", event.Data.ID, decoded.Data.ID)
	}
	if decoded.Data.Email != event.Data.Email {
		t.Errorf("Data.Email: expected %q, got %q", event.Data.Email, decoded.Data.Email)
	}
}
