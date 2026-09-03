package channel

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"shared/models"
	"shared/repositories/notifications"
	"time"
)

var (
	ErrTransient = errors.New("transient delivery error")
	ErrPermanent = errors.New("permanent delivery error")
)

type Sender interface {
	Send(ctx context.Context, n notifications.NotificationRow) error
}

func NewEmailSender(logger *slog.Logger) Sender {
	return NewFakeSender("email", logger)
}
func NewPushSender(logger *slog.Logger) Sender {
	return NewFakeSender("push", logger)
}
func NewSMSSender(logger *slog.Logger) Sender {
	return NewFakeSender("sms", logger)
}

type FakeSender struct {
	channel models.NotificationChannel
	logger  *slog.Logger
}

func NewFakeSender(channel models.NotificationChannel, logger *slog.Logger) *FakeSender {
	return &FakeSender{
		channel: channel,
		logger:  logger,
	}
}

func (s *FakeSender) Send(ctx context.Context, n notifications.NotificationRow) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	// детерминированно: 1% permanent, 10% transient
	h := fnv.New32a()
	_, _ = fmt.Fprintf(h, "%s:%s:%d", n.EventID, n.Channel, n.Attempts)
	mod := h.Sum32() % 100
	switch {
	case mod < 1:
		return ErrPermanent
	case mod < 11:
		return ErrTransient
	}
	s.logger.Info("notification sent",
		"user_id", n.UserID,
		"event_id", n.EventID,
		"channel", n.Channel,
		"attempt", n.Attempts+1,
	)
	// имитация сети с отменой по ctx
	timer := time.NewTimer(20 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
