package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"awesomeProject/pkg/middleware"
	"awesomeProject/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// EventPublisher defines the interface for publishing events.
type EventPublisher interface {
	Publish(routingKey string, body []byte, correlationID string) error
}

// UserHandler handles user-related HTTP requests.
type UserHandler struct {
	DB        *sql.DB
	Publisher EventPublisher
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(db *sql.DB, pub EventPublisher) *UserHandler {
	return &UserHandler{DB: db, Publisher: pub}
}

// CreateUser godoc
// @Summary      Create a new user
// @Description  Creates a new user and publishes a user.created event
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request  body      models.CreateUserRequest  true  "Create user request"
// @Success      201      {object}  models.User
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	correlationID := middleware.GetCorrelationID(c)
	log.Printf("[API] CreateUser correlation_id=%s", correlationID)

	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := models.User{
		ID:        uuid.New().String(),
		Email:     req.Email,
		Name:      req.Name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Insert into database
	_, err := h.DB.Exec(
		"INSERT INTO users (id, email, name, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)",
		user.ID, user.Email, user.Name, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		log.Printf("[API] Error creating user: %v correlation_id=%s", err, correlationID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// Publish event
	event := models.UserEvent{
		EventID:       uuid.New().String(),
		CorrelationID: correlationID,
		EventType:     models.EventUserCreated,
		Timestamp:     time.Now(),
		Data:          user,
	}

	eventBytes, _ := json.Marshal(event)
	if err := h.Publisher.Publish(string(models.EventUserCreated), eventBytes, correlationID); err != nil {
		log.Printf("[API] Error publishing event: %v correlation_id=%s", err, correlationID)
		// Don't fail the request — event will be missed but user is created
	}

	log.Printf("[API] User created: id=%s email=%s correlation_id=%s", user.ID, user.Email, correlationID)
	c.JSON(http.StatusCreated, user)
}

// UpdateUser godoc
// @Summary      Update an existing user
// @Description  Updates a user and publishes a user.updated event
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id       path      string                    true  "User ID"
// @Param        request  body      models.UpdateUserRequest  true  "Update user request"
// @Success      200      {object}  models.User
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	correlationID := middleware.GetCorrelationID(c)
	userID := c.Param("id")
	log.Printf("[API] UpdateUser id=%s correlation_id=%s", userID, correlationID)

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current user
	var user models.User
	err := h.DB.QueryRow("SELECT id, email, name, created_at, updated_at FROM users WHERE id = $1", userID).
		Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
		return
	}

	// Apply updates
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Name != "" {
		user.Name = req.Name
	}
	user.UpdatedAt = time.Now()

	// Update in database
	_, err = h.DB.Exec(
		"UPDATE users SET email = $1, name = $2, updated_at = $3 WHERE id = $4",
		user.Email, user.Name, user.UpdatedAt, user.ID,
	)
	if err != nil {
		log.Printf("[API] Error updating user: %v correlation_id=%s", err, correlationID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	// Publish event
	event := models.UserEvent{
		EventID:       uuid.New().String(),
		CorrelationID: correlationID,
		EventType:     models.EventUserUpdated,
		Timestamp:     time.Now(),
		Data:          user,
	}

	eventBytes, _ := json.Marshal(event)
	if err := h.Publisher.Publish(string(models.EventUserUpdated), eventBytes, correlationID); err != nil {
		log.Printf("[API] Error publishing event: %v correlation_id=%s", err, correlationID)
	}

	log.Printf("[API] User updated: id=%s correlation_id=%s", user.ID, correlationID)
	c.JSON(http.StatusOK, user)
}

// GetUser godoc
// @Summary      Get a user by ID
// @Description  Returns a single user
// @Tags         users
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  models.User
// @Failure      404  {object}  map[string]string
// @Router       /users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	err := h.DB.QueryRow("SELECT id, email, name, created_at, updated_at FROM users WHERE id = $1", userID).
		Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// ListUsers godoc
// @Summary      List all users
// @Description  Returns all users
// @Tags         users
// @Produce      json
// @Success      200  {array}   models.User
// @Failure      500  {object}  map[string]string
// @Router       /users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	rows, err := h.DB.Query("SELECT id, email, name, created_at, updated_at FROM users ORDER BY created_at DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt, &u.UpdatedAt); err != nil {
			continue
		}
		users = append(users, u)
	}

	if users == nil {
		users = []models.User{}
	}

	c.JSON(http.StatusOK, users)
}
