package consumer

import (
	"context"
	"fmt"
	"log/slog"
	"notification-ingestor/internal/ingest"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader    *kafka.Reader
	processor *ingest.Processor
	logger    *slog.Logger
}

func NewConsumer(
	reader *kafka.Reader,
	processor *ingest.Processor,
	logger *slog.Logger,
) *Consumer {
	return &Consumer{
		reader:    reader,
		processor: processor,
		logger:    logger,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	const op = "consumer.Run"

	for {
		// graceful shutdown
		if err := ctx.Err(); err != nil {
			return nil
		}

		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("%s: fetch: %w", op, err)
		}

		commit, err := c.processor.Process(ctx, msg)
		if err != nil {
			c.logger.Error(
				"process message failed",
				"topic", msg.Topic,
				"partition", msg.Partition,
				"offset", msg.Offset,
				"err", err,
			)
			continue
		}

		if commit {
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				return fmt.Errorf("%s: commit: %w", op, err)
			}
		}

	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
