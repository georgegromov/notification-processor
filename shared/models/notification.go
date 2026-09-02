package models

import (
	"time"

	"github.com/google/uuid"
)

type NotificationChannel string

const (
	NotificationChannelEmail NotificationChannel = "email"
	NotificationChannelSMS   NotificationChannel = "sms"
	NotificationChannelPush  NotificationChannel = "push"
)

var notificationChannelDistributionByEventType = map[EventType][]NotificationChannel{
	EventTypeOrderCreated:    {NotificationChannelEmail, NotificationChannelPush},
	EventTypePaymentReceived: {NotificationChannelEmail},
	EventTypeOrderShipped:    {NotificationChannelSMS, NotificationChannelPush},
}

type NotificationStatus string

const (
	NotificationStatusPending    NotificationStatus = "pending"
	NotificationStatusProcessing NotificationStatus = "processing"
	NotificationStatusSent       NotificationStatus = "sent"
	NotificationStatusFailed     NotificationStatus = "failed"
)

type Notification struct {
	ID          uuid.UUID           `json:"id"`
	EventID     uuid.UUID           `json:"event_id"`
	UserID      uuid.UUID           `json:"user_id"`
	Channel     NotificationChannel `json:"channel"`
	Status      NotificationStatus  `json:"status"`
	Attempts    int                 `json:"attempts"`
	NextRetryAt time.Time           `json:"next_retry_at"`
	LockedAt    time.Time           `json:"locked_at"`
	LastError   string              `json:"last_error"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}
