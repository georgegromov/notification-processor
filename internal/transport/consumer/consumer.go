package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"notification-processor/internal/application"
	"notification-processor/internal/domain"
)

// Consumer читает события из Kafka и передаёт их в WorkerPool.
//
// Ответственность:
//  1. Чтение сообщений из топика
//  2. Десериализация eventDTO → domain.Event
//  3. Poison pill - битое сообщение логируем, коммитим offset, продолжаем
//  4. Ручной коммит offset - только после подтверждения от WorkerPool
type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, groupID, topic string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: groupID,

		// Откуда читать при первом запуске
		StartOffset: kafka.FirstOffset,

		// Отключаем автокоммит - коммитим вручную после обработки
		CommitInterval: 0,

		// Минимум и максимум данных за один fetch
		MinBytes: 1,
		MaxBytes: 10e6, // 10 MB

		// Таймаут ожидания новых сообщений
		MaxWait: 500 * time.Millisecond,

		// Logger
		Logger:      kafka.LoggerFunc(func(msg string, args ...interface{}) { slog.Debug(fmt.Sprintf(msg, args...)) }),
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) { slog.Error(fmt.Sprintf(msg, args...)) }),
	})

	slog.Info("kafka consumer ready", "topic", topic, "group_id", groupID, "brokers", brokers)
	return &Consumer{reader: reader}
}

const (
	CommitOffsetTimeout = 5 * time.Second
)

// Run — основной цикл чтения. Блокируется до отмены ctx.
//
// На каждое сообщение создаёт application.Job с функцией CommitOffset
// и передаёт в pool. WorkerPool вызовет CommitOffset после обработки.
func (c *Consumer) Run(ctx context.Context, pool *application.WorkerPool) {
	slog.Info("consumer loop started")
	defer slog.Info("consumer loop stopped")

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("kafka fetch error", "error", err)
			time.Sleep(time.Second)
			continue
		}

		event, err := parseMessage(msg)
		if err != nil {
			slog.Warn("poison pill, skipping", "partition", msg.Partition, "offset", msg.Offset, "error", err, "value", string(msg.Value))
			commitCtx, commitCancel := context.WithTimeout(context.Background(), CommitOffsetTimeout)
			if commitErr := c.reader.CommitMessages(commitCtx, msg); commitErr != nil {
				slog.Error("failed to commit poison pill offset", "error", commitErr)
			}
			commitCancel()
			continue
		}

		captured := msg
		job := application.NewJob(event, func() error {
			commitCtx, cancel := context.WithTimeout(context.Background(), CommitOffsetTimeout)
			defer cancel()

			if err := c.reader.CommitMessages(commitCtx, captured); err != nil {
				return fmt.Errorf("commit offset %d: %w", captured.Offset, err)
			}
			slog.Debug("offset committed", "partition", captured.Partition, "offset", captured.Offset)
			return nil
		},
		)

		if !pool.Submit(ctx, *job) {
			return
		}
	}
}

// Close корректно завершает consumer.
func (c *Consumer) Close() {
	slog.Info("closing kafka consumer")
	if err := c.reader.Close(); err != nil {
		slog.Error("error closing kafka reader", "error", err)
	}
}

// eventDTO → domain.Event
type eventDTO struct {
	EventID   string         `json:"event_id"`
	UserID    int64          `json:"user_id"`
	EventType string         `json:"event_type"`
	Timestamp time.Time      `json:"timestamp"`
	Payload   map[string]any `json:"payload"`
}

func parseMessage(msg kafka.Message) (domain.Event, error) {
	var dto eventDTO
	if err := json.Unmarshal(msg.Value, &dto); err != nil {
		return domain.Event{}, fmt.Errorf("json unmarshal: %w", err)
	}

	eventID, err := uuid.Parse(dto.EventID)
	if err != nil {
		return domain.Event{}, fmt.Errorf("invalid event_id %q: %w", dto.EventID, err)
	}

	event := domain.NewEvent(
		domain.EventID(eventID.String()),
		domain.UserID(dto.UserID),
		domain.EventType(dto.EventType),
		dto.Timestamp,
		dto.Payload,
	)

	if err := event.Validate(); err != nil {
		return domain.Event{}, fmt.Errorf("validation: %w", err)
	}

	return event, nil
}
