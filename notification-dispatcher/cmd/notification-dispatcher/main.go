package main

import (
	"context"
	"log/slog"
	"notification-dispatcher/internal/breaker"
	"notification-dispatcher/internal/channel"
	"notification-dispatcher/internal/dispatcher"
	"os"
	"os/signal"
	"shared/db"
	"shared/models"
	"shared/repositories/notifications"
	"sync"
	"syscall"
	"time"
)

const (
	dsn = "postgres://postgres:postgres@localhost:5432/notification_processor_db?sslmode=disable"

	breakerFailureThreshold = 5
	breakerCooldown         = 10 * time.Second
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
		return
	}
	defer dbClient.Close()

	repo := notifications.NewRepository(dbClient.GetPool())

	type channelSetup struct {
		name   models.NotificationChannel
		sender channel.Sender
	}

	channels := []channelSetup{
		{models.NotificationChannelEmail, channel.NewEmailSender(logger)},
		{models.NotificationChannelPush, channel.NewPushSender(logger)},
		{models.NotificationChannelSMS, channel.NewSMSSender(logger)},
	}

	var wg sync.WaitGroup
	for _, ch := range channels {
		wg.Add(1)
		w := dispatcher.NewWorker(
			ch.name,
			repo,
			ch.sender,
			breaker.New(breakerFailureThreshold, breakerCooldown),
			logger.With("channel", ch.name),
		)
		go func() {
			defer wg.Done()
			if err := w.Run(ctx); err != nil {
				logger.Error("worker stopped", "channel", ch.name, "err", err)
			}
		}()
	}

	logger.Info("notification-dispatcher started")
	<-ctx.Done()
	wg.Wait()
	logger.Info("notification-dispatcher stopped")

}
