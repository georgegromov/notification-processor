package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"notification-processor/internal/application"
	"notification-processor/internal/config"
	"notification-processor/internal/repository"
	"notification-processor/internal/transport/consumer"
	"notification-processor/internal/transport/emulator"
	"notification-processor/pkg/db"
)

func main() {
	// Логирование
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("starting notification-processor")

	// Конфигурация
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Контекст — отменяется по SIGTERM/SIGINT
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		sig := <-sigCh
		slog.Info("received shutdown signal", "signal", sig)
		cancel()
	}()

	// PostgreSQL
	pool, err := db.NewPool(ctx, cfg.DB)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	schemaSQL, err := os.ReadFile(db.SchemaPath)
	if err != nil {
		slog.Error("failed to read schema", "path", db.SchemaPath, "error", err)
		os.Exit(1)
	}
	if err := db.ApplySchema(ctx, pool, string(schemaSQL)); err != nil {
		slog.Error("failed to apply schema", "error", err)
		os.Exit(1)
	}

	// DI
	repo := repository.NewRepository(pool)
	emulator := emulator.NewEmulator(cfg.RetryMaxAttempts, cfg.RetryBaseDelay)
	uc := application.NewProcessEventUseCase(repo, emulator)
	workerPool := application.NewWorkerPool(cfg.WorkerCount, uc)

	consumer := consumer.NewConsumer(cfg.KafkaBrokers, cfg.KafkaGroupID, cfg.KafkaTopic)

	// WorkerPool стартует в фоне — ждёт задачи из канала
	go workerPool.Start(ctx)

	// Consumer блокируется до отмены ctx (SIGTERM)
	consumer.Run(ctx, workerPool)

	// ctx уже отменён — consumer.Run вышел
	// Говорим пулу: новых задач не будет, дообработай текущие
	slog.Info("shutting down, waiting for workers to finish")
	workerPool.Stop()
	workerPool.Wait()

	consumer.Close()
	slog.Info("notification-processor stopped")
}
