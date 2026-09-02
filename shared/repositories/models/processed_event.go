package models

import (
	"time"

	"github.com/google/uuid"
)

type ProcessedEventRecord struct {
	EventID   uuid.UUID `db:"event_id"`
	Topic     string    `db:"topic"`
	Partition int       `db:"partition"`
	Offset    int64     `db:"offset"`
	CreatedAt time.Time `db:"created_at"`
}
