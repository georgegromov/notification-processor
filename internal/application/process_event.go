package application

import (
	"context"
	"errors"
	"log/slog"
	"notification-processor/internal/domain"
	"time"
)

var (
	// ErrAlreadyProcessed — дубликат события, безопасно пропустить.
	ErrAlreadyProcessed = errors.New("event already processed")

	// ErrUnknownEventType — неизвестный тип события, безопасно пропустить.
	ErrUnknownEventType = errors.New("unknown event type")
)

var (
	ExecuteTimeout = time.Second * 60
)

type EventRepository interface {
	SaveEventTx(ctx context.Context, event domain.Event, notifications []domain.Notification) (alreadyProcessed bool, err error)
	UpdateNotificationStatus(
		ctx context.Context,
		eventID domain.EventID,
		channel domain.Channel,
		status domain.NotificationStatus,
		attempts int,
		lastErr string,
	) error
}

// DeliveryResult — результат попытки доставки уведомления.
type DeliveryResult struct {
	Attempts int
	// nil = успех, TempError или PermError = сбой
	Err error
}

type Deliverer interface {
	Send(ctx context.Context, notification domain.Notification) DeliveryResult
}

// ProcessEventUseCase — use case сервиса
//
// Алгоритм:
//  1. BuildNotifications — роутинг: event -> []Notification
//  2. SaveEventTx        — dedup + сохранение в одной транзакции
//  3. Send               — доставка по каждому каналу
//  4. UpdateStatus       — фиксируем результат в БД
type ProcessEventUseCase struct {
	repository EventRepository
	deliverer  Deliverer
}

func NewProcessEventUseCase(repo EventRepository, deliverer Deliverer) *ProcessEventUseCase {
	return &ProcessEventUseCase{
		repository: repo,
		deliverer:  deliverer,
	}
}

// Execute обрабатывает одно событие.
// Возвращает:
//   - nil                  — успех, offset нужно закоммитить
//   - ErrAlreadyProcessed  — дубликат, offset нужно закоммитить
//   - ErrUnknownEventType  — неизвестный тип, offset нужно закоммитить
//   - любая другая ошибка  — инфраструктурный сбой (БД), offset НЕ коммитить
func (uc *ProcessEventUseCase) Execute(ctx context.Context, event domain.Event) error {
	ctx, cancel := context.WithTimeout(ctx, ExecuteTimeout)
	defer cancel()

	log := slog.With(
		"event_id", event.EventID,
		"user_id", event.UserID,
		"event_type", event.EventType,
	)

	// Шаг 1: роутинг
	notifications, err := domain.BuildNotifications(event)
	if err != nil {
		log.Warn("unknown event type, skipping", "error", err)
		return ErrUnknownEventType
	}

	// Шаг 2: dedup + сохранение в БД
	alreadyProcessed, err := uc.repository.SaveEventTx(ctx, event, notifications)
	if err != nil {
		log.Error("failed to save event to db", "error", err)
		return err
	}
	if alreadyProcessed {
		log.Info("duplicate event, skipping")
		return ErrAlreadyProcessed
	}

	log.Info("event saved, delivering notifications", "count", len(notifications))

	// Доставка и обновление статуса
	for _, n := range notifications {
		uc.deliver(ctx, n)
	}

	return nil
}

// deliver — доставляет одно уведомление и сохраняет результат в БД.
// Не возвращает ошибку: сбой доставки одного канала не должен останавливать доставку по остальным каналам.
func (uc *ProcessEventUseCase) deliver(ctx context.Context, n domain.Notification) {
	log := slog.With("event_id", n.EventID, "user_id", n.UserID, "channel", n.Channel)

	result := uc.deliverer.Send(ctx, n)
	status, lastErr := resolveStatus(result.Err)

	if err := uc.repository.UpdateNotificationStatus(
		ctx, n.EventID, n.Channel, status, result.Attempts, lastErr,
	); err != nil {
		log.Error("failed to update notification status", "status", status, "error", err)
		return
	}

	log.Info("notification processed", "status", status, "attempts", result.Attempts)
}

// resolveStatus переводит ошибку доставки в статус для БД.
func resolveStatus(err error) (domain.NotificationStatus, string) {
	switch {
	case err == nil:
		return domain.NotificationStatusDelivered, ""
	case domain.IsPermError(err):
		return domain.NotificationStatusFailedPerm, err.Error()
	default:
		// TempError после исчерпания всех retry в Deliverer
		return domain.NotificationStatusFailedTemp, err.Error()
	}
}
