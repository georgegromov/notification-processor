package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type client struct {
	pool *pgxpool.Pool
}

func NewClient(ctx context.Context, dsn string) (*client, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return &client{pool: pool}, nil
}

func (c *client) GetPool() *pgxpool.Pool {
	return c.pool
}

func (c *client) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

func (c *client) Close() {
	c.pool.Close()
}
