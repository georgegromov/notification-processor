package models

import (
	"time"

	"github.com/google/uuid"
)

type ProcessedEventRecord struct {
	EventID   uuid.UUID `db:"event_id"`
	Topic     string    `db:"kafka_topic"`
	Partition int       `db:"kafka_partition"`
	Offset    int64     `db:"kafka_offset"`
	CreatedAt time.Time `db:"created_at"`
}
