package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"notification-ingestor/internal/dlq"
	"notification-ingestor/internal/router"
	"shared/models"
	"shared/repositories/processed_events"
	"shared/validation"
	"time"

	"github.com/google/uuid"
	kgo "github.com/segmentio/kafka-go"
)

// DTO входящего Kafka-сообщения (user_id = int64, как в task и migrations)
type kafkaEvent struct {
	EventID   uuid.UUID        `json:"event_id" validate:"required,uuid"`
	UserID    uuid.UUID        `json:"user_id" validate:"required,uuid"`
	EventType models.EventType `json:"event_type" validate:"required"`
	Timestamp time.Time        `json:"timestamp" validate:"required"`
	Payload   json.RawMessage  `json:"payload"`
}

type Processor struct {
	validator *validation.Service
	router    *router.Router
	inbox     processed_events.Repository
	dlq       *dlq.DLQ
	logger    *slog.Logger
}

func NewProcessor(
	validator *validation.Service,
	router *router.Router,
	inbox processed_events.Repository,
	dlq *dlq.DLQ,
	logger *slog.Logger,
) *Processor {
	return &Processor{
		validator: validator,
		router:    router,
		inbox:     inbox,
		dlq:       dlq,
		logger:    logger,
	}
}

func (p *Processor) Process(ctx context.Context, msg kgo.Message) (commit bool, err error) {
	const op = "ingest.processor.Process"

	// 1. Decode msg
	// 2. Validate msg
	// 3. Route msg to appropriate channels
	// 4. Insert into inbox + outbox

	var event kafkaEvent
	if err = json.Unmarshal(msg.Value, &event); err != nil {
		if dlqErr := p.dlq.Write(
			ctx,
			msg.Topic,
			msg.Partition,
			msg.Offset,
			msg.Key,
			msg.Value,
			"invalid json",
		); dlqErr != nil {
			return false, fmt.Errorf("%s: dlq after json error: %w", op, dlqErr)
		}
		p.logger.Error(op, slog.String("error", err.Error()))
		return true, err
	}

	if err := p.validator.Validate(ctx, event); err != nil {
		if dlqErr := p.dlq.Write(
			ctx,
			msg.Topic,
			msg.Partition,
			msg.Offset,
			msg.Key,
			msg.Value,
			err.Error(),
		); dlqErr != nil {
			return false, fmt.Errorf("%s: dlq after validation error: %w", op, dlqErr)
		}
		p.logger.Warn("poison pill: validation failed", "event_id", event.EventID, "err", err)
		return true, nil
	}

	channels, err := p.router.ChannelsFor(event.EventType)
	if err != nil {
		if dlqErr := p.dlq.Write(
			ctx,
			msg.Topic,
			msg.Partition,
			msg.Offset,
			msg.Key,
			msg.Value,
			err.Error(),
		); dlqErr != nil {
			return false, fmt.Errorf("%s: dlq after routing error: %w", op, dlqErr)
		}
		p.logger.Warn("poison pill: unknown event_type", "event_id", event.EventID, "event_type", event.EventType)
		return true, nil
	}

	notifications := make([]processed_events.NotificationInput, 0, len(channels))
	for _, ch := range channels {

		notificationInput := processed_events.NotificationInput{
			EventID: event.EventID,
			UserID:  event.UserID,
			Channel: ch,
		}

		notifications = append(notifications, notificationInput)
	}

	eventInput := processed_events.ProcessedEventInput{
		EventID:   event.EventID,
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
	}

	inserted, err := p.inbox.InsertTx(ctx, eventInput, notifications)
	if err != nil {
		return false, fmt.Errorf("%s: insert tx: %w", op, err)
	}

	if !inserted {
		p.logger.Info("duplicate event skipped", "event_id", event.EventID)
	}

	return true, nil
}
