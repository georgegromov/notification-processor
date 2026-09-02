package kafka

import (
	kgo "github.com/segmentio/kafka-go"
)

type writer struct {
	writer *kgo.Writer
}

func NewWriter(brokers []string) *kgo.Writer {
	return &kgo.Writer{
		Addr:                   kgo.TCP(brokers...),
		Balancer:               &kgo.Hash{},
		AllowAutoTopicCreation: true,
	}
}
