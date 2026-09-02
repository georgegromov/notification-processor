package processed_events

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"shared/models"
)

type ProcessedEventInput struct {
	EventID   uuid.UUID
	Topic     string
	Partition int
	Offset    int64
}

type NotificationInput struct {
	EventID uuid.UUID
	UserID  uuid.UUID
	Channel models.NotificationChannel
}

type Repository interface {
	InsertTx(ctx context.Context, event ProcessedEventInput, notifications []NotificationInput) (inserted bool, err error)
}

type repository struct {
	pool    *pgxpool.Pool
	queries *queries
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{
		pool:    pool,
		queries: mustLoadQueries(),
	}
}

func (r *repository) InsertTx(
	ctx context.Context,
	event ProcessedEventInput,
	notifications []NotificationInput,
) (bool, error) {
	const op = "processed_events.repository.InsertTx"

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	tag, err := tx.Exec(ctx, r.queries.insertProcessedEvent,
		event.EventID, event.Topic, event.Partition, event.Offset,
	)
	if err != nil {
		return false, fmt.Errorf("%s: insert processed event: %w", op, err)
	}
	if tag.RowsAffected() == 0 {
		// дубликат event_id — outbox не трогаем
		if err = tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("%s: commit duplicate: %w", op, err)
		}
		return false, nil
	}

	batch := &pgx.Batch{}
	for _, n := range notifications {
		batch.Queue(
			r.queries.insertNotification,
			n.EventID, n.UserID, string(n.Channel),
		)
	}

	results := tx.SendBatch(ctx, batch)
	for i := 0; i < batch.Len(); i++ {
		if _, err = results.Exec(); err != nil {
			results.Close()
			return false, fmt.Errorf("%s: insert notification #%d: %w", op, i, err)
		}
	}
	results.Close()

	if err = tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("%s: commit: %w", op, err)
	}
	return true, nil
}
