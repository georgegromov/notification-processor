package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func main() {
	brokers := getEnvOrDefault("KAFKA_BROKERS", "localhost:9092")
	topic := getEnvOrDefault("KAFKA_TOPIC", "user-events")
	count := getEnvInt("EVENT_COUNT", 100)

	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		// Async=false — ждём подтверждения от брокера
		Async: false,
	}
	defer writer.Close()

	slog.Info("producing test events", "count", count, "topic", topic)

	ctx := context.Background()

	// Фиксируем один event_id для проверки идемпотентности
	duplicateID := uuid.New().String()
	eventTypes := []string{"order_created", "payment_received", "order_shipped"}

	for i := range count {
		var value []byte
		var label string

		switch {
		// Каждое 10-е — poison pill (битый JSON)
		case i%10 == 9:
			value = []byte(`{"event_id": "broken", "user_id": }`)
			label = "poison_pill"

		// i=4 — дубликат i=0 (проверяем идемпотентность)
		case i == 4:
			value = newEvent(duplicateID, int64(i%50+1), "order_created",
				map[string]any{"note": "duplicate of i=0", "order_id": 1000})
			label = "duplicate"

		// i=0 — оригинал который будет продублирован
		case i == 0:
			value = newEvent(duplicateID, int64(i%50+1), "order_created",
				map[string]any{"order_id": 1000})
			label = "order_created (original, will be duplicated at i=4)"

		default:
			eventType := eventTypes[i%len(eventTypes)]
			value = newEvent(uuid.New().String(), int64(i%50+1), eventType,
				map[string]any{"order_id": 1000 + i, "amount": (i + 1) * 100})
			label = eventType
		}

		err := writer.WriteMessages(ctx, kafka.Message{Value: value})
		if err != nil {
			slog.Error("produce failed", "i", i, "error", err)
			continue
		}

		slog.Info("produced", "i", i, "label", label)
		time.Sleep(100 * time.Millisecond)
	}

	slog.Info("producer done")
}

type eventDTO struct {
	EventID   string         `json:"event_id"`
	UserID    int64          `json:"user_id"`
	EventType string         `json:"event_type"`
	Timestamp time.Time      `json:"timestamp"`
	Payload   map[string]any `json:"payload"`
}

func newEvent(eventID string, userID int64, eventType string, payload map[string]any) []byte {
	e := eventDTO{
		EventID:   eventID,
		UserID:    userID,
		EventType: eventType,
		Timestamp: time.Now(),
		Payload:   payload,
	}
	b, _ := json.Marshal(e)
	return b
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
