package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

const (
	topic = "users-events"
)

var brokers = []string{"kafka:29092"}

type Event struct {
	EventID   uuid.UUID      `json:"event_id"`
	EventType string         `json:"event_type"`
	UserID    uuid.UUID      `json:"user_id"`
	Timestamp time.Time      `json:"timestamp"`
	Payload   map[string]any `json:"payload"`
}

func main() {
	ctx := context.Background()

	w := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireAll,
		Async:                  false,
	}
	defer w.Close()

	userID := uuid.New()

	// 1. Нормальный поток: все 3 event_type
	normal := []Event{
		newEvent(userID, "order_created", map[string]any{"order_id": "ORD-1"}),
		newEvent(userID, "payment_received", map[string]any{"amount": 1500}),
		newEvent(userID, "order_shipped", map[string]any{"tracking": "TRACK-1"}),
	}
	mustWrite(ctx, w, normal...)
	log.Printf("sent %d normal events", len(normal))

	// 2. Дубликат event_id — повторной отправки быть не должно
	dup := normal[0]
	mustWrite(ctx, w, dup)
	log.Printf("sent duplicate event_id=%s", dup.EventID)

	// 3. Poison pills
	mustWriteRaw(ctx, w, []byte(userID.String()), []byte(`{not-json`))
	log.Printf("sent poison: invalid json")
	mustWriteRaw(ctx, w, []byte(userID.String()), mustJSON(map[string]any{
		"event_id":   uuid.Nil.String(), // пустой/нулевой uuid — валидация должна отсечь
		"user_id":    userID.String(),
		"event_type": "order_created",
		"timestamp":  time.Now().Format(time.RFC3339),
		"payload":    map[string]any{},
	}))
	log.Printf("sent poison: empty event_id")
	mustWrite(ctx, w, newEvent(userID, "unknown_type", map[string]any{"x": 1}))
	log.Printf("sent poison: unknown event_type")

	// 4. Ещё пачка для нагрузки/демо
	batch := make([]Event, 0, 20)
	for i := 0; i < 20; i++ {
		types := []string{"order_created", "payment_received", "order_shipped"}
		batch = append(batch, newEvent(uuid.New(), types[i%3], map[string]any{
			"n": i,
		}))
	}

	mustWrite(ctx, w, batch...)
	log.Printf("sent batch=%d", len(batch))
	log.Println("done")
}

func newEvent(userID uuid.UUID, eventType string, payload map[string]any) Event {
	return Event{
		EventID:   uuid.New(),
		UserID:    userID,
		EventType: eventType,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}
}

func mustWrite(ctx context.Context, w *kafka.Writer, events ...Event) {
	msgs := make([]kafka.Message, 0, len(events))
	for _, e := range events {
		msgs = append(msgs, kafka.Message{
			Key:   []byte(e.UserID.String()),
			Value: mustJSON(e),
			Time:  time.Now(),
		})
	}

	if err := w.WriteMessages(ctx, msgs...); err != nil {
		log.Fatalf("write: %v", err)
	}
}

func mustWriteRaw(ctx context.Context, w *kafka.Writer, key, value []byte) {
	if err := w.WriteMessages(ctx, kafka.Message{
		Key:   key,
		Value: value,
		Time:  time.Now(),
	}); err != nil {
		log.Fatalf("write raw: %v", err)
	}
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Errorf("marshal: %w", err))
	}
	return b
}
