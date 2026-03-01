package domain

import (
	"errors"
	"time"
)

var (
	ErrMissingEventID   = errors.New("missing event id")
	ErrMissingUserID    = errors.New("missing user id")
	ErrMissingEventType = errors.New("missing event type")
	ErrMissingTimestamp = errors.New("missing timestamp")
)

// EventID - уникальный идентификатор события
type EventID string

// UserID - уникальный идентификатор пользователя
type UserID int64

// Тип события
type EventType string

const (
	EventOrderCreated    EventType = "order_created"
	EventPaymentReceived EventType = "payment_received"
	EventOrderShipped    EventType = "order_shipped"
)

// Event — входящее сообщение из Kafka
// Пример JSON:
//
//	{
//	  "event_id":"550e8400-e29b-41d4-a716-446655440000",
//	  "user_id":12345,
//	  "event_type":"order_created",
//	  "timestamp":"2026-03-01T15:30:00Z",
//	  "payload":{ "order_id": 99, "amount": 1500 }
//	}
type Event struct {
	EventID   EventID
	UserID    UserID
	EventType EventType
	Timestamp time.Time
	Payload   map[string]any
}

// NewEvent создаёт событие с заданными полями. Payload может быть nil.
func NewEvent(eventID EventID, userID UserID, eventType EventType, timestamp time.Time, payload map[string]any) Event {
	return Event{
		EventID:   eventID,
		UserID:    userID,
		EventType: eventType,
		Timestamp: timestamp,
		Payload:   payload,
	}
}

// Validate проверяет что обязательные поля заполнены.
// Вызывается после json.Unmarshal чтобы поймать семантически битые сообщения
// (JSON валидный, но смысла нет).
func (e *Event) Validate() error {
	if e.EventID == "" {
		return ErrMissingEventID
	}
	if e.UserID == 0 {
		return ErrMissingUserID
	}
	if e.EventType == "" {
		return ErrMissingEventType
	}
	if e.Timestamp.IsZero() {
		return ErrMissingTimestamp
	}
	return nil
}
