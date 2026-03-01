// Тесты счётчика попыток Attempts.
// NewAttempts — начальное состояние: 0, CanRetry, не Exhausted.
// IncrementAndExhaust — Increment до лимита, CanRetry/Exhausted.
// SingleAttempt — лимит 1, одна попытка, Exhausted.
// ValueSemantics — Increment возвращает новое значение, исходный не меняется.
package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"notification-processor/internal/domain"
)

func TestAttempts_NewAttempts(t *testing.T) {
	a := domain.NewAttempts(3)
	assert.Equal(t, 0, a.Current())
	assert.True(t, a.CanRetry())
	assert.False(t, a.Exhausted())
}

func TestAttempts_IncrementAndExhaust(t *testing.T) {
	a := domain.NewAttempts(3)

	a = a.Increment()
	require.Equal(t, 1, a.Current())
	assert.True(t, a.CanRetry())
	assert.False(t, a.Exhausted())

	a = a.Increment()
	require.Equal(t, 2, a.Current())
	assert.True(t, a.CanRetry())

	a = a.Increment()
	require.Equal(t, 3, a.Current())
	assert.False(t, a.CanRetry())
	assert.True(t, a.Exhausted())
}

func TestAttempts_SingleAttempt(t *testing.T) {
	a := domain.NewAttempts(1)
	assert.True(t, a.CanRetry())
	a = a.Increment()
	assert.Equal(t, 1, a.Current())
	assert.False(t, a.CanRetry())
	assert.True(t, a.Exhausted())
}

func TestAttempts_ValueSemantics(t *testing.T) {
	a := domain.NewAttempts(2)
	a1 := a.Increment()
	a2 := a1.Increment()
	assert.Equal(t, 0, a.Current())
	assert.Equal(t, 1, a1.Current())
	assert.Equal(t, 2, a2.Current())
}
