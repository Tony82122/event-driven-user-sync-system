package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUserJSON(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	user := User{
		ID:        "usr-001",
		Email:     "jane@example.com",
		Name:      "Jane Doe",
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("failed to marshal User: %v", err)
	}

	var decoded User
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal User: %v", err)
	}

	if decoded.ID != user.ID {
		t.Errorf("ID: expected %q, got %q", user.ID, decoded.ID)
	}
	if decoded.Email != user.Email {
		t.Errorf("Email: expected %q, got %q", user.Email, decoded.Email)
	}
	if decoded.Name != user.Name {
		t.Errorf("Name: expected %q, got %q", user.Name, decoded.Name)
	}
}

func TestCreateUserRequestJSON(t *testing.T) {
	input := `{"email":"bob@example.com","name":"Bob Smith"}`
	var req CreateUserRequest
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("failed to unmarshal CreateUserRequest: %v", err)
	}
	if req.Email != "bob@example.com" {
		t.Errorf("Email: expected %q, got %q", "bob@example.com", req.Email)
	}
	if req.Name != "Bob Smith" {
		t.Errorf("Name: expected %q, got %q", "Bob Smith", req.Name)
	}
}

func TestUpdateUserRequestJSON(t *testing.T) {
	input := `{"email":"new@example.com"}`
	var req UpdateUserRequest
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("failed to unmarshal UpdateUserRequest: %v", err)
	}
	if req.Email != "new@example.com" {
		t.Errorf("Email: expected %q, got %q", "new@example.com", req.Email)
	}
	if req.Name != "" {
		t.Errorf("Name: expected empty, got %q", req.Name)
	}
}
