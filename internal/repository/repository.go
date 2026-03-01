package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"notification-processor/internal/domain"
)

const (
	SaveEventTxTimeout  = 15 * time.Second
	UpdateStatusTimeout = 10 * time.Second
)

// repository реализует application.EventRepository.
// Две гарантии которые обеспечивает этот слой:
//  1. Идемпотентность — повторный event_id не создаёт дубликат (ON CONFLICT DO NOTHING)
//  2. Атомарность — processed_events + notifications в одной транзакции
type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *repository {
	return &repository{pool: pool}
}

// SaveEventTx — атомарно создает событие и создаёт уведомления.
//
// Транзакция:
//  1. INSERT INTO processed_events ON CONFLICT DO NOTHING
//     -> 0 rows -> возвращаем alreadyProcessed=true
//  2. Batch INSERT INTO notifications для каждого канала
//  3. COMMIT
//
// Если сервис упадёт после COMMIT но до коммита kafka offset —
// при повторной обработке вернется alreadyProcessed=true.
func (r *repository) SaveEventTx(
	ctx context.Context,
	event domain.Event,
	notifications []domain.Notification,
) (alreadyProcessed bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, SaveEventTxTimeout)
	defer cancel()

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	q := `INSERT INTO processed_events (event_id) VALUES ($1) ON CONFLICT DO NOTHING`

	tag, err := tx.Exec(ctx, q, event.EventID)
	if err != nil {
		return false, fmt.Errorf("insert processed_event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return true, nil
	}

	// pgx.Batch отправляет все запросы одним походом к БД.
	// Для order_created (2 канала) — один сетевой вызов вместо двух.
	batch := &pgx.Batch{}
	q = `INSERT INTO notifications (event_id, user_id, channel, status) VALUES ($1, $2, $3, $4)`
	for _, n := range notifications {
		batch.Queue(q, n.EventID, n.UserID, string(n.Channel), string(domain.NotificationStatusPending))
	}

	res := tx.SendBatch(ctx, batch)
	for range notifications {
		if _, err = res.Exec(); err != nil {
			_ = res.Close()
			return false, fmt.Errorf("insert notification: %w", err)
		}
	}

	// Освобождаем ресурсы и flush результаты
	if err = res.Close(); err != nil {
		return false, fmt.Errorf("batch close: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit tx: %w", err)
	}

	return false, nil
}

// UpdateNotificationStatus — обновляет статус и число попыток после доставки.
func (r *repository) UpdateNotificationStatus(ctx context.Context, eventID domain.EventID, channel domain.Channel, status domain.NotificationStatus, attempts int, lastErr string) error {
	ctx, cancel := context.WithTimeout(ctx, UpdateStatusTimeout)
	defer cancel()

	q := `
    UPDATE notifications 
      SET status=$1, attempts=$2, last_error=NULLIF($3, ''), updated_at = now() 
    WHERE event_id = $4 AND channel = $5
  `

	tag, err := r.pool.Exec(ctx, q, string(status), attempts, lastErr, eventID, string(channel))
	if err != nil {
		return fmt.Errorf("update notification status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("notification not found: event_id=%s channel=%s", eventID, channel)
	}
	return nil
}

var _ interface {
	SaveEventTx(ctx context.Context, event domain.Event, notifications []domain.Notification) (alreadyProcessed bool, err error)
	UpdateNotificationStatus(ctx context.Context, eventID domain.EventID, channel domain.Channel, status domain.NotificationStatus, attempts int, lastErr string) error
} = (*repository)(nil)
