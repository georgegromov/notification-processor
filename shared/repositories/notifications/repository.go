package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"shared/models"
)

type NotificationRow struct {
	ID          int64
	EventID     uuid.UUID
	UserID      uuid.UUID
	Channel     models.NotificationChannel
	Status      models.NotificationStatus
	Attempts    int
	NextRetryAt time.Time
	LockedAt    *time.Time
	LastError   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Repository interface {
	ClaimPending(ctx context.Context, channel models.NotificationChannel, limit int) ([]NotificationRow, error)
	MarkSent(ctx context.Context, id int64) error
	ScheduleRetry(ctx context.Context, id int64, attempts int, nextRetryAt time.Time, lastError string) error
	MarkFailed(ctx context.Context, id int64, attempts int, lastError string) error
	ReclaimStale(ctx context.Context, leaseSeconds int) (int64, error)
}

type repository struct {
	pool    *pgxpool.Pool
	queries *queries
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool, queries: mustLoadQueries()}
}

func (r *repository) ClaimPending(ctx context.Context, channel models.NotificationChannel, limit int) ([]NotificationRow, error) {
	rows, err := r.pool.Query(ctx, r.queries.claimPending, string(channel), limit)
	if err != nil {
		return nil, fmt.Errorf("claim pending: %w", err)
	}
	defer rows.Close()

	var out []NotificationRow
	for rows.Next() {
		var n NotificationRow
		var ch, status string
		if err := rows.Scan(
			&n.ID, &n.EventID, &n.UserID, &ch, &status,
			&n.Attempts, &n.NextRetryAt, &n.LockedAt, &n.LastError,
			&n.CreatedAt, &n.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		n.Channel = models.NotificationChannel(ch)
		n.Status = models.NotificationStatus(status)
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *repository) MarkSent(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, r.queries.markSent, id)
	return err
}

func (r *repository) ScheduleRetry(ctx context.Context, id int64, attempts int, nextRetryAt time.Time, lastError string) error {
	_, err := r.pool.Exec(ctx, r.queries.scheduleRetry, id, attempts, nextRetryAt, lastError)
	return err
}

func (r *repository) MarkFailed(ctx context.Context, id int64, attempts int, lastError string) error {
	_, err := r.pool.Exec(ctx, r.queries.markFailed, id, attempts, lastError)
	return err
}

func (r *repository) ReclaimStale(ctx context.Context, leaseSeconds int) (int64, error) {
	tag, err := r.pool.Exec(ctx, r.queries.reclaimStale, leaseSeconds)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
