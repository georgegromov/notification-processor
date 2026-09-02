package kafka

import (
	"fmt"
	"log/slog"
	"time"

	kgo "github.com/segmentio/kafka-go"
)

func NewReader(brokers []string, groupID, topic string) *kgo.Reader {
	return kgo.NewReader(kgo.ReaderConfig{
		Brokers:        brokers,
		GroupID:        groupID,
		Topic:          topic,
		StartOffset:    kgo.FirstOffset,
		CommitInterval: 0,

		// Batching logic
		MinBytes: 1,                      // default — вернуть сразу, как есть сообщение
		MaxBytes: 10e6,                   // default 1MB — для JSON событий достаточно
		MaxWait:  500 * time.Millisecond, // ждать батч до 500ms (default 10s — долго для low-latency)

		// Consumer group logic
		HeartbeatInterval: 3 * time.Second,  // default 3s — обычно ок
		SessionTimeout:    30 * time.Second, // default 10s — мало, если обработка + TX в Postgres > 10s

		// Retry logic
		ReadBackoffMin: 100 * time.Millisecond,
		ReadBackoffMax: 1 * time.Second,
		MaxAttempts:    3,

		Dialer: &kgo.Dialer{
			Timeout:   10 * time.Second,
			DualStack: true,
		},

		Logger:      kgo.LoggerFunc(func(msg string, args ...any) { slog.Debug(fmt.Sprintf(msg, args...)) }),
		ErrorLogger: kgo.LoggerFunc(func(msg string, args ...any) { slog.Error(fmt.Sprintf(msg, args...)) }),
	})
}
