package emulator

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"time"

	"notification-processor/internal/application"
	"notification-processor/internal/domain"
)

// emulator реализует application.Deliverer.
//
// Эмулирует внешний сервис доставки:
//
//	89% → успех
//	10% → TempError: ретраи с экспоненциальной задержкой
//	1% → PermError: ретраи до 3 попыток, после 3-й прекращаем
type emulator struct {
	maxAttempts int
	baseDelay   time.Duration
	randFunc    func() float64 // если задан, используется вместо rand.Float64 (для тестов)
}

// Option настраивает emulator (например, для тестов).
type Option func(*emulator)

// WithRandFunc задаёт источник случайности. Используется в тестах для детерминированного поведения.
func WithRandFunc(f func() float64) Option {
	return func(e *emulator) { e.randFunc = f }
}

func NewEmulator(maxAttempts int, baseDelay time.Duration, opts ...Option) *emulator {
	e := &emulator{
		maxAttempts: maxAttempts,
		baseDelay:   baseDelay,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Send доставляет уведомление с retry логикой.
// Временные ошибки: ретраи с экспоненциальной задержкой (baseDelay * 2^0, 2^1, …).
// Перманентные ошибки: ретраи до 3 попыток, после 3-й прекращаем.
func (e *emulator) Send(ctx context.Context, n domain.Notification) application.DeliveryResult {
	a := domain.NewAttempts(e.maxAttempts)

	for a.CanRetry() {
		a = a.Increment()
		attemptNum := a.Current()

		err := e.callExternalService(n)

		if err == nil {
			slog.Info("notification delivered", "user_id", n.UserID, "channel", n.Channel, "event_type", n.EventType, "attempt", attemptNum)
			return application.DeliveryResult{Attempts: attemptNum}
		}

		// Любая ошибка (временная или перманентная): логируем и проверяем лимит попыток
		if domain.IsPermError(err) {
			slog.Error("permanent delivery failure", "user_id", n.UserID, "channel", n.Channel, "attempt", attemptNum, "max_attempts", e.maxAttempts, "error", err)
		} else {
			slog.Warn("temporary delivery failure", "user_id", n.UserID, "channel", n.Channel, "attempt", attemptNum, "max_attempts", e.maxAttempts, "error", err)
		}

		// После 3-й попытки прекращаем (и для Temp, и для Perm)
		if a.Exhausted() {
			return application.DeliveryResult{Attempts: attemptNum, Err: err}
		}

		backoff := e.getBackoff(attemptNum)
		slog.Debug("retrying after backoff", "channel", n.Channel, "backoff", backoff, "next_attempt", attemptNum+1)

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return application.DeliveryResult{
				Attempts: attemptNum,
				Err:      fmt.Errorf("delivery cancelled: %w", ctx.Err()),
			}
		}
	}

	panic("unreachable")
}

// callExternalService эмулирует один вызов внешнего API.
//
// Распределение:
//
//	[0.00, 0.01) -> PermError  (1%)
//	[0.01, 0.11) -> TempError  (10%)
//	[0.11, 1.00) -> nil        (89%)
func (e *emulator) callExternalService(n domain.Notification) error {
	r := rand.Float64()
	if e.randFunc != nil {
		r = e.randFunc()
	}
	switch {
	case r < 0.01:
		return &domain.PermError{
			Msg: fmt.Sprintf("user %d is permanently unreachable via %s", n.UserID, n.Channel),
		}
	case r < 0.11:
		return &domain.TempError{
			Msg: fmt.Sprintf("%s service temporarily unavailable", n.Channel),
		}
	default:
		return nil
	}
}

// getBackoff - вычисляет экспоненциальную задержку: baseDelay * 2^(attemptNum-1)
func (e *emulator) getBackoff(attemptNum int) time.Duration {
	return Backoff(e.baseDelay, attemptNum)
}

// Backoff вычисляет экспоненциальную задержку для попытки attemptNum: baseDelay * 2^(attemptNum-1).
// Экспортируется для тестов и переиспользования.
func Backoff(baseDelay time.Duration, attemptNum int) time.Duration {
	return time.Duration(float64(baseDelay) * math.Pow(2, float64(attemptNum-1)))
}

// Compile-time проверка что Emulator реализует application.Deliverer.
var _ application.Deliverer = (*emulator)(nil)
