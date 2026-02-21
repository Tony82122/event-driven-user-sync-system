package crm

import (
	"encoding/json"
	"testing"
	"time"

	"awesomeProject/pkg/models"

	"github.com/DATA-DOG/go-sqlmock"
	amqp "github.com/rabbitmq/amqp091-go"
)

func makeDelivery(event models.UserEvent) amqp.Delivery {
	body, _ := json.Marshal(event)
	return amqp.Delivery{
		Body:          body,
		CorrelationId: event.CorrelationID,
		RoutingKey:    string(event.EventType),
	}
}

func TestHandleMessage_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	consumer := NewConsumer(db)
	consumer.SimulateFailures = false

	event := models.UserEvent{
		EventID:       "evt-001",
		CorrelationID: "corr-001",
		EventType:     models.EventUserCreated,
		Timestamp:     time.Now(),
		Data: models.User{
			ID:    "user-001",
			Email: "test@example.com",
			Name:  "Test User",
		},
	}

	// Idempotency check — not a duplicate
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("evt-001").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// CRM sync log insert
	mock.ExpectExec("INSERT INTO crm_sync_log").
		WithArgs("evt-001", "corr-001", "user.created", "user-001", "test@example.com", "Test User").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Idempotency key insert
	mock.ExpectExec("INSERT INTO idempotency_keys").
		WithArgs("evt-001").
		WillReturnResult(sqlmock.NewResult(1, 1))

	delivery := makeDelivery(event)
	if err := consumer.HandleMessage(delivery); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestHandleMessage_DuplicateEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	consumer := NewConsumer(db)
	consumer.SimulateFailures = false

	event := models.UserEvent{
		EventID:       "evt-dup",
		CorrelationID: "corr-dup",
		EventType:     models.EventUserUpdated,
		Timestamp:     time.Now(),
		Data: models.User{
			ID:    "user-002",
			Email: "dup@example.com",
			Name:  "Dup User",
		},
	}

	// Idempotency check — IS a duplicate
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("evt-dup").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	delivery := makeDelivery(event)
	if err := consumer.HandleMessage(delivery); err != nil {
		t.Fatalf("expected no error for duplicate, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestHandleMessage_InvalidJSON(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	consumer := NewConsumer(db)
	consumer.SimulateFailures = false

	delivery := amqp.Delivery{
		Body:          []byte("{invalid json"),
		CorrelationId: "corr-bad",
	}

	if err := consumer.HandleMessage(delivery); err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
