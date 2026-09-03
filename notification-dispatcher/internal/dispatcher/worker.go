package dispatcher

import (
	"context"
	"errors"
	"log/slog"
	"notification-dispatcher/internal/breaker"
	"notification-dispatcher/internal/channel"
	"notification-dispatcher/internal/retry"
	"shared/models"
	"shared/repositories/notifications"
	"time"
)

const (
	DefaultBatchSize    = 10
	DefaultPollInterval = 300 * time.Millisecond
	DefaultSendTimeout  = 2 * time.Second
	DefaultMaxAttempts  = 3
	DefaultLeaseSeconds = 30
)

type Worker struct {
	channel      models.NotificationChannel
	repo         notifications.Repository
	sender       channel.Sender
	breaker      *breaker.Breaker
	logger       *slog.Logger
	batchSize    int
	pollInterval time.Duration
	sendTimeout  time.Duration
	maxAttempts  int
	leaseSeconds int
}

func NewWorker(
	ch models.NotificationChannel,
	repo notifications.Repository,
	sender channel.Sender,
	br *breaker.Breaker,
	logger *slog.Logger,
) *Worker {
	return &Worker{
		channel:      ch,
		repo:         repo,
		sender:       sender,
		breaker:      br,
		logger:       logger,
		batchSize:    DefaultBatchSize,
		pollInterval: DefaultPollInterval,
		sendTimeout:  DefaultSendTimeout,
		maxAttempts:  DefaultMaxAttempts,
		leaseSeconds: DefaultLeaseSeconds,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	reclaimTicker := time.NewTicker(10 * time.Second)
	defer reclaimTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-reclaimTicker.C:
			if n, err := w.repo.ReclaimStale(ctx, w.leaseSeconds); err != nil {
				w.logger.Error("reclaim stale failed", "channel", w.channel, "err", err)
			} else if n > 0 {
				w.logger.Info("reclaimed stale", "channel", w.channel, "count", n)
			}
		case <-ticker.C:
			if err := w.poll(ctx); err != nil {
				w.logger.Error("poll failed", "channel", w.channel, "err", err)
			}
		}
	}
}

func (w *Worker) poll(ctx context.Context) error {
	if !w.breaker.Allow() {
		return nil
	}
	items, err := w.repo.ClaimPending(ctx, w.channel, w.batchSize)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := w.handle(ctx, item); err != nil {
			w.logger.Error("handle failed", "id", item.ID, "err", err)
		}
	}
	return nil
}

func (w *Worker) handle(ctx context.Context, n notifications.NotificationRow) error {
	sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), w.sendTimeout)
	defer cancel()
	err := w.sender.Send(sendCtx, n)
	attempts := n.Attempts + 1
	switch {
	case err == nil:
		w.breaker.Success()
		return w.repo.MarkSent(ctx, n.ID)
	case errors.Is(err, channel.ErrPermanent):
		w.breaker.Failure()
		return w.repo.MarkFailed(ctx, n.ID, attempts, err.Error())
	case errors.Is(err, channel.ErrTransient):
		w.breaker.Failure()
		if attempts >= w.maxAttempts {
			return w.repo.MarkFailed(ctx, n.ID, attempts, err.Error())
		}
		next := time.Now().Add(retry.Next(attempts))
		return w.repo.ScheduleRetry(ctx, n.ID, attempts, next, err.Error())
	default:
		// timeout / unknown → как transient
		w.breaker.Failure()
		if attempts >= w.maxAttempts {
			return w.repo.MarkFailed(ctx, n.ID, attempts, err.Error())
		}
		next := time.Now().Add(retry.Next(attempts))
		return w.repo.ScheduleRetry(ctx, n.ID, attempts, next, err.Error())
	}
}
