package main

import (
	"context"
	"log/slog"
	"notification-ingestor/internal/consumer"
	"notification-ingestor/internal/dlq"
	"notification-ingestor/internal/ingest"
	"notification-ingestor/internal/router"
	"os"
	"os/signal"
	"shared/db"
	"shared/kafka"
	"shared/repositories/processed_events"
	"shared/validation"
	"syscall"
)

const (
	dsn             = "postgres://postgres:postgres@postgres:5432/notification_processor_db?sslmode=disable"
	consumerGroupID = "notification-ingestor"
	eventsTopic     = "users-events"
	dlqTopic        = "users-events-dlq"
)

var (
	brokers = []string{"kafka:29092"}
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	dbClient, err := db.NewClient(ctx, dsn)
	if err != nil {
		logger.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer dbClient.Close()

	reader := kafka.NewReader(brokers, consumerGroupID, eventsTopic)
	writer := kafka.NewWriter(brokers)
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
	}()

	processor := ingest.NewProcessor(
		validation.NewService(),
		router.New(),
		processed_events.NewRepository(dbClient.GetPool()),
		dlq.New(dlqTopic, writer),
		logger,
	)

	c := consumer.NewConsumer(reader, processor, logger)

	logger.Info(
		"notification-ingestor started",
		"brokers", brokers,
		"topic", eventsTopic,
		"consumer_group_id", consumerGroupID,
	)
	if err := c.Run(ctx); err != nil {
		logger.Error("consumer stopped with error", "err", err)
		return
	}

	logger.Info("notification-ingestor stopped")
}
