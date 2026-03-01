// Тесты эмулятора доставки: retry, backoff и отмена по контексту.
// Backoff — формула задержки baseDelay*2^(attemptNum-1) для попыток 1..4.
// Send_SuccessFirstTry — успех с первой попытки.
// Send_TempErrorThenSuccess — два TempError, затем успех на 3-й попытке.
// Send_TempErrorExhausted_ReturnsErr — все попытки TempError, лимит 3, ошибка.
// Send_PermError_ReturnsImmediatelyAfterAttempt — PermError на каждой попытке, 3 попытки.
// Send_ContextCanceled_DuringBackoff — отмена контекста во время backoff, ошибка cancelled.
// Send_OneAttemptMax_NoRetry — maxAttempts=1 и TempError, одна попытка без ретраев.
package emulator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"notification-processor/internal/domain"
)

func TestBackoff(t *testing.T) {
	base := 10 * time.Millisecond
	assert.Equal(t, 10*time.Millisecond, Backoff(base, 1))
	assert.Equal(t, 20*time.Millisecond, Backoff(base, 2))
	assert.Equal(t, 40*time.Millisecond, Backoff(base, 3))
	assert.Equal(t, 80*time.Millisecond, Backoff(base, 4))
}

func TestSend_SuccessFirstTry(t *testing.T) {
	// r >= 0.11 → success
	callCount := 0
	e := NewEmulator(3, time.Millisecond, WithRandFunc(func() float64 {
		callCount++
		return 0.5
	}))
	n := domain.Notification{UserID: 1, Channel: domain.ChannelEmail, EventType: domain.EventOrderCreated}

	result := e.Send(context.Background(), n)

	require.Nil(t, result.Err)
	assert.Equal(t, 1, result.Attempts)
	assert.Equal(t, 1, callCount)
}

func TestSend_TempErrorThenSuccess(t *testing.T) {
	// 0.01 <= r < 0.11 → TempError. Возвращаем Temp два раза, потом success.
	calls := 0
	e := NewEmulator(3, 1*time.Millisecond, WithRandFunc(func() float64 {
		calls++
		if calls <= 2 {
			return 0.05 // TempError
		}
		return 0.5 // success
	}))
	n := domain.Notification{UserID: 1, Channel: domain.ChannelEmail, EventType: domain.EventOrderCreated}

	result := e.Send(context.Background(), n)

	require.NoError(t, result.Err)
	assert.Equal(t, 3, result.Attempts)
	assert.Equal(t, 3, calls)
}

func TestSend_TempErrorExhausted_ReturnsErr(t *testing.T) {
	// Все попытки — TempError, лимит 3
	e := NewEmulator(3, 1*time.Millisecond, WithRandFunc(func() float64 {
		// всегда TempError
		return 0.05
	}))
	n := domain.Notification{UserID: 1, Channel: domain.ChannelEmail, EventType: domain.EventOrderCreated}

	result := e.Send(context.Background(), n)

	require.Error(t, result.Err)
	assert.True(t, domain.IsTempError(result.Err))
	assert.Equal(t, 3, result.Attempts)
}

func TestSend_PermError_ReturnsImmediatelyAfterAttempt(t *testing.T) {
	// PermError тоже ретраится до maxAttempts (3), затем возвращаем ошибку.
	callCount := 0
	e := NewEmulator(3, 1*time.Millisecond, WithRandFunc(func() float64 {
		callCount++
		// PermError
		return 0.0
	}))
	n := domain.Notification{UserID: 1, Channel: domain.ChannelEmail, EventType: domain.EventOrderCreated}

	result := e.Send(context.Background(), n)

	require.Error(t, result.Err)
	assert.True(t, domain.IsPermError(result.Err))
	assert.Equal(t, 3, result.Attempts)
	assert.Equal(t, 3, callCount)
}

func TestSend_ContextCanceled_DuringBackoff(t *testing.T) {
	// Первая попытка — TempError, во время backoff отменяем контекст
	e := NewEmulator(3, 5*time.Second, WithRandFunc(func() float64 {
		return 0.05 // TempError
	}))
	n := domain.Notification{UserID: 1, Channel: domain.ChannelEmail, EventType: domain.EventOrderCreated}
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	result := e.Send(ctx, n)

	require.Error(t, result.Err)
	assert.Contains(t, result.Err.Error(), "cancelled")
	assert.Equal(t, 1, result.Attempts)
}

func TestSend_OneAttemptMax_NoRetry(t *testing.T) {
	callCount := 0
	e := NewEmulator(1, time.Second, WithRandFunc(func() float64 {
		callCount++
		return 0.05 // TempError
	}))
	n := domain.Notification{UserID: 1, Channel: domain.ChannelEmail, EventType: domain.EventOrderCreated}

	result := e.Send(context.Background(), n)

	require.Error(t, result.Err)
	assert.Equal(t, 1, result.Attempts)
	assert.Equal(t, 1, callCount)
}
