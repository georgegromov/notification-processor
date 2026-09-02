// Package dlq provides a dead letter queue implementation for Kafka.
package dlq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type DLQRecord struct {
	Topic     string    `json:"topic"`
	Partition int       `json:"partition"`
	Offset    int64     `json:"offset"`
	Key       string    `json:"key,omitempty"`
	Value     []byte    `json:"value"`
	Error     string    `json:"error"`
	Time      time.Time `json:"time"`
}

type DLQ struct {
	topic  string
	writer *kafka.Writer
}

func New(topic string, writer *kafka.Writer) *DLQ {
	return &DLQ{topic: topic, writer: writer}
}

func (d *DLQ) Write(
	ctx context.Context,
	topic string,
	partition int,
	offset int64,
	key []byte,
	value []byte,
	reason string,
) error {

	now := time.Now()

	rec := DLQRecord{
		Topic:     topic,
		Partition: partition,
		Offset:    offset,
		Key:       string(key),
		Value:     value,
		Error:     reason,
		Time:      now,
	}

	payload, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("dlq marshal: %w", err)
	}

	if err := d.writer.WriteMessages(ctx, kafka.Message{
		Topic: d.topic,
		Key:   key,
		Value: payload,
		Time:  now,
	}); err != nil {
		return fmt.Errorf("dlq write: %w", err)
	}

	return nil
}
