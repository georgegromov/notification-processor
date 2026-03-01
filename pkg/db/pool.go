package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const SchemaPath = "migrations/schema.sql"

// ApplySchema применяет SQL схему к базе данных.
func ApplySchema(ctx context.Context, pool *pgxpool.Pool, schemaSQL string) error {
	if _, err := pool.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	slog.Info("schema applied")
	return nil
}

type Config struct {
	DSN      string
	MaxConns int32
	MinConns int32
}

// NewPool создаёт pgxpool с retry при старте.
func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}

	const maxAttempts = 10
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				slog.Info("connected to postgres", "attempt", attempt)
				return pool, nil
			} else {
				pool.Close()
				err = pingErr
			}
		}

		if attempt == maxAttempts {
			return nil, fmt.Errorf("postgres not ready after %d attempts: %w", maxAttempts, err)
		}

		delay := time.Duration(attempt) * time.Second
		slog.Warn("postgres not ready, retrying", "attempt", attempt, "delay", delay, "error", err)

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	panic("unreachable")
}
