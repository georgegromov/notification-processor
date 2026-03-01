// Тесты WorkerPool (очередь задач, коммит offset по результату).
// Submit_ProcessesJobAndCommits — задача обработана, CommitOffset вызван.
// Submit_ContextCanceled_ReturnsFalse — контекст отменён при полном буфере, Submit возвращает false.
// InfraError_DoesNotCommit — инфраструктурная ошибка use case, CommitOffset не вызывается.
// ErrAlreadyProcessed_Commits — дубликат события (ErrAlreadyProcessed), CommitOffset вызывается.
package application_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"notification-processor/internal/application"
	"notification-processor/internal/domain"
)

// mockPoolUsecase реализует usecase для тестов пула
type mockPoolUsecase struct {
	mu        sync.Mutex
	calls     []domain.Event
	returnErr error // если не nil, Execute возвращает эту ошибку
}

func (m *mockPoolUsecase) Execute(_ context.Context, event domain.Event) error {
	m.mu.Lock()
	m.calls = append(m.calls, event)
	err := m.returnErr
	m.mu.Unlock()
	return err
}

func (m *mockPoolUsecase) getCalls() []domain.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]domain.Event(nil), m.calls...)
}

func TestWorkerPool_Submit_ProcessesJobAndCommits(t *testing.T) {
	uc := &mockPoolUsecase{}
	pool := application.NewWorkerPool(2, uc)
	ctx := context.Background()

	go pool.Start(ctx)
	defer func() {
		pool.Stop()
		pool.Wait()
	}()

	event := domain.NewEvent(domain.EventID(uuid.New().String()), 100, domain.EventOrderCreated, time.Now(), nil)
	committed := false
	job := application.NewJob(event, func() error {
		committed = true
		return nil
	})

	ok := pool.Submit(ctx, *job)
	require.True(t, ok)

	// Даём воркеру время обработать
	time.Sleep(100 * time.Millisecond)

	calls := uc.getCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, event.EventID, calls[0].EventID)
	assert.True(t, committed, "CommitOffset should have been called")
}

func TestWorkerPool_Submit_ContextCanceled_ReturnsFalse(t *testing.T) {
	uc := &mockPoolUsecase{}
	pool := application.NewWorkerPool(1, uc) // буфер канала = 2
	ctx, cancel := context.WithCancel(context.Background())
	go pool.Start(ctx)
	defer func() {
		pool.Stop()
		pool.Wait()
	}()
	// Заполняем канал (1 воркер + буфер 2 = 3 слота), чтобы следующий Submit заблокировался
	for i := 0; i < 3; i++ {
		event := domain.NewEvent(domain.EventID(uuid.New().String()), 1, domain.EventOrderCreated, time.Now(), nil)
		pool.Submit(ctx, *application.NewJob(event, func() error { return nil }))
	}
	// Этот Submit блокируется; отменяем контекст — должен вернуть false
	cancel()
	event := domain.NewEvent(domain.EventID(uuid.New().String()), 1, domain.EventOrderCreated, time.Now(), nil)
	ok := pool.Submit(ctx, *application.NewJob(event, func() error { return nil }))
	assert.False(t, ok, "Submit must return false when context is canceled")
}

func TestWorkerPool_InfraError_DoesNotCommit(t *testing.T) {
	infraErr := errors.New("db connection lost")
	uc := &mockPoolUsecase{returnErr: infraErr}
	pool := application.NewWorkerPool(1, uc)
	ctx := context.Background()

	go pool.Start(ctx)
	defer func() {
		pool.Stop()
		pool.Wait()
	}()

	event := domain.NewEvent(domain.EventID(uuid.New().String()), 1, domain.EventOrderCreated, time.Now(), nil)
	committed := false
	job := application.NewJob(event, func() error {
		committed = true
		return nil
	})

	ok := pool.Submit(ctx, *job)
	require.True(t, ok)
	time.Sleep(100 * time.Millisecond)

	assert.False(t, committed, "CommitOffset must not be called on infra error")
}

func TestWorkerPool_ErrAlreadyProcessed_Commits(t *testing.T) {
	uc := &mockPoolUsecase{returnErr: application.ErrAlreadyProcessed}
	pool := application.NewWorkerPool(1, uc)
	ctx := context.Background()

	go pool.Start(ctx)
	defer func() {
		pool.Stop()
		pool.Wait()
	}()

	committed := false
	job := application.NewJob(domain.NewEvent(domain.EventID(uuid.New().String()), 1, domain.EventOrderCreated, time.Now(), nil), func() error {
		committed = true
		return nil
	})

	ok := pool.Submit(ctx, *job)
	require.True(t, ok)
	time.Sleep(100 * time.Millisecond)

	assert.True(t, committed, "CommitOffset must be called for ErrAlreadyProcessed")
}
