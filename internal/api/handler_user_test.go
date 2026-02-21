package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"awesomeProject/pkg/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockPublisher implements EventPublisher for testing.
type mockPublisher struct {
	published []publishedMsg
	err       error
}

type publishedMsg struct {
	RoutingKey    string
	Body          []byte
	CorrelationID string
}

func (m *mockPublisher) Publish(routingKey string, body []byte, correlationID string) error {
	m.published = append(m.published, publishedMsg{
		RoutingKey:    routingKey,
		Body:          body,
		CorrelationID: correlationID,
	})
	return m.err
}

func TestCreateUser_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO users").
		WithArgs(sqlmock.AnyArg(), "test@example.com", "Test User", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	pub := &mockPublisher{}
	handler := NewUserHandler(db, pub)
	router := NewRouter(handler)

	body := `{"email":"test@example.com","name":"Test User"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var user models.User
	if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", user.Email)
	}
	if user.Name != "Test User" {
		t.Errorf("expected name Test User, got %s", user.Name)
	}
	if user.ID == "" {
		t.Error("expected user ID to be set")
	}

	// Verify event was published
	if len(pub.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(pub.published))
	}
	if pub.published[0].RoutingKey != "user.created" {
		t.Errorf("expected routing key user.created, got %s", pub.published[0].RoutingKey)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestCreateUser_BadRequest(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	pub := &mockPublisher{}
	handler := NewUserHandler(db, pub)
	router := NewRouter(handler)

	// Missing required fields
	body := `{"email":"not-an-email"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateUser_InvalidJSON(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	pub := &mockPublisher{}
	handler := NewUserHandler(db, pub)
	router := NewRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/users", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestGetUser_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "email", "name", "created_at", "updated_at"}).
		AddRow("user-123", "test@example.com", "Test User", now, now)
	mock.ExpectQuery("SELECT id, email, name, created_at, updated_at FROM users WHERE id = \\$1").
		WithArgs("user-123").
		WillReturnRows(rows)

	pub := &mockPublisher{}
	handler := NewUserHandler(db, pub)
	router := NewRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/user-123", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var user models.User
	if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if user.ID != "user-123" {
		t.Errorf("expected ID user-123, got %s", user.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "email", "name", "created_at", "updated_at"})
	mock.ExpectQuery("SELECT id, email, name, created_at, updated_at FROM users WHERE id = \\$1").
		WithArgs("nonexistent").
		WillReturnRows(rows)

	pub := &mockPublisher{}
	handler := NewUserHandler(db, pub)
	router := NewRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/nonexistent", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListUsers_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "email", "name", "created_at", "updated_at"}).
		AddRow("user-1", "one@example.com", "User One", now, now).
		AddRow("user-2", "two@example.com", "User Two", now, now)
	mock.ExpectQuery("SELECT id, email, name, created_at, updated_at FROM users ORDER BY created_at DESC").
		WillReturnRows(rows)

	pub := &mockPublisher{}
	handler := NewUserHandler(db, pub)
	router := NewRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var users []models.User
	if err := json.Unmarshal(w.Body.Bytes(), &users); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestListUsers_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "email", "name", "created_at", "updated_at"})
	mock.ExpectQuery("SELECT id, email, name, created_at, updated_at FROM users ORDER BY created_at DESC").
		WillReturnRows(rows)

	pub := &mockPublisher{}
	handler := NewUserHandler(db, pub)
	router := NewRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var users []models.User
	if err := json.Unmarshal(w.Body.Bytes(), &users); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}

func TestUpdateUser_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	now := time.Now()
	selectRows := sqlmock.NewRows([]string{"id", "email", "name", "created_at", "updated_at"}).
		AddRow("user-123", "old@example.com", "Old Name", now, now)
	mock.ExpectQuery("SELECT id, email, name, created_at, updated_at FROM users WHERE id = \\$1").
		WithArgs("user-123").
		WillReturnRows(selectRows)

	mock.ExpectExec("UPDATE users SET email = \\$1, name = \\$2, updated_at = \\$3 WHERE id = \\$4").
		WithArgs("new@example.com", "New Name", sqlmock.AnyArg(), "user-123").
		WillReturnResult(sqlmock.NewResult(0, 1))

	pub := &mockPublisher{}
	handler := NewUserHandler(db, pub)
	router := NewRouter(handler)

	body := `{"email":"new@example.com","name":"New Name"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/users/user-123", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var user models.User
	if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if user.Email != "new@example.com" {
		t.Errorf("expected email new@example.com, got %s", user.Email)
	}

	// Verify event was published
	if len(pub.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(pub.published))
	}
	if pub.published[0].RoutingKey != "user.updated" {
		t.Errorf("expected routing key user.updated, got %s", pub.published[0].RoutingKey)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestUpdateUser_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "email", "name", "created_at", "updated_at"})
	mock.ExpectQuery("SELECT id, email, name, created_at, updated_at FROM users WHERE id = \\$1").
		WithArgs("nonexistent").
		WillReturnRows(rows)

	pub := &mockPublisher{}
	handler := NewUserHandler(db, pub)
	router := NewRouter(handler)

	body := `{"name":"Updated"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/users/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHealthCheck(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	pub := &mockPublisher{}
	handler := NewUserHandler(db, pub)
	router := NewRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
}

func TestCorrelationIDPassedToEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO users").
		WithArgs(sqlmock.AnyArg(), "corr@example.com", "Corr Test", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	pub := &mockPublisher{}
	handler := NewUserHandler(db, pub)
	router := NewRouter(handler)

	body := `{"email":"corr@example.com","name":"Corr Test"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Correlation-ID", "test-corr-id-123")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify correlation ID was passed to publisher
	if len(pub.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(pub.published))
	}
	if pub.published[0].CorrelationID != "test-corr-id-123" {
		t.Errorf("expected correlation ID test-corr-id-123, got %s", pub.published[0].CorrelationID)
	}

	// Verify correlation ID in the event body
	var event models.UserEvent
	if err := json.Unmarshal(pub.published[0].Body, &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}
	if event.CorrelationID != "test-corr-id-123" {
		t.Errorf("expected event correlation ID test-corr-id-123, got %s", event.CorrelationID)
	}
}
