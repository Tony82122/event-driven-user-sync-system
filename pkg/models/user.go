package models

import "time"

// User represents a user in the system.
type User struct {
	ID        string    `json:"id" db:"id"`
	Email     string    `json:"email" db:"email" binding:"required,email"`
	Name      string    `json:"name" db:"name" binding:"required"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CreateUserRequest is the request body for creating a user.
type CreateUserRequest struct {
	Email string `json:"email" binding:"required,email" example:"john@example.com"`
	Name  string `json:"name" binding:"required" example:"John Doe"`
}

// UpdateUserRequest is the request body for updating a user.
type UpdateUserRequest struct {
	Email string `json:"email,omitempty" binding:"omitempty,email" example:"john@example.com"`
	Name  string `json:"name,omitempty" binding:"omitempty" example:"John Doe"`
}
