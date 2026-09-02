package models

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTypeOrderCreated    EventType = "order_created"
	EventTypePaymentReceived EventType = "payment_received"
	EventTypeOrderShipped    EventType = "order_shipped"
)

type Event struct {
	EventID   uuid.UUID `json:"event_id"`
	EventType EventType `json:"event_type"`
	UserID    uuid.UUID `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload"`
}
