// Тесты use case ProcessEvent (обработка одного события).
// OrderCreated_SendsEmailAndPush — order_created, отправка email и push.
// PaymentReceived_SendsEmailOnly — payment_received, только email.
// OrderShipped_SendsPushAndSMS — order_shipped, push и sms.
// DuplicateEvent_SkipsDelivery — уже обработано, доставка не вызывается.
// UnknownEventType_SkipsDelivery — неизвестный event_type, SaveEventTx не вызывается.
// DBError_ReturnsErrorWithoutDelivery — ошибка БД, доставки нет.
// TempDeliveryError_StatusFailedTemp — доставка вернула TempError, статус failed_temp.
// PermDeliveryError_StatusFailedPerm — доставка вернула PermError, статус failed_perm.
// OneChannelFails_OtherChannelStillDelivered — один канал failed_perm, второй delivered.
package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"notification-processor/internal/application"
	"notification-processor/internal/domain"
)

// =============================================================================
// Моки — простые структуры, без mockgen и других генераторов.
// Go стиль: минимальные моки рядом с тестами.
// =============================================================================

// mockRepo реализует application.EventRepository
type mockRepo struct {
	// Что возвращать из SaveEventTx
	alreadyProcessed bool
	saveErr          error

	// Что возвращать из UpdateNotificationStatus
	updateErr error

	// Записываем вызовы для проверки в тестах
	savedEvent         domain.Event
	savedNotifications []domain.Notification
	updatedStatuses    []updatedStatus
}

type updatedStatus struct {
	channel  domain.Channel
	status   domain.NotificationStatus
	attempts int
	lastErr  string
}

func (m *mockRepo) SaveEventTx(_ context.Context, event domain.Event, notifications []domain.Notification) (bool, error) {
	m.savedEvent = event
	m.savedNotifications = notifications
	return m.alreadyProcessed, m.saveErr
}

func (m *mockRepo) UpdateNotificationStatus(_ context.Context, _ domain.EventID, channel domain.Channel, status domain.NotificationStatus, attempts int, lastErr string) error {
	m.updatedStatuses = append(m.updatedStatuses, updatedStatus{channel, status, attempts, lastErr})
	return m.updateErr
}

// mockDeliverer реализует application.Deliverer
type mockDeliverer struct {
	// Можно задать разный результат для каждого канала
	results map[domain.Channel]application.DeliveryResult
	// Записываем что было вызвано
	sent []domain.Notification
}

func (m *mockDeliverer) Send(_ context.Context, n domain.Notification) application.DeliveryResult {
	m.sent = append(m.sent, n)
	if r, ok := m.results[n.Channel]; ok {
		return r
	}
	// По умолчанию — успех
	return application.DeliveryResult{Attempts: 1}
}

// =============================================================================
// Helpers
// =============================================================================

func makeEvent(eventType domain.EventType) domain.Event {
	return domain.NewEvent(domain.EventID(uuid.New().String()), 42, eventType, time.Now(), nil)
}

func newUseCase(repo *mockRepo, deliverer *mockDeliverer) *application.ProcessEventUseCase {
	return application.NewProcessEventUseCase(repo, deliverer)
}

// =============================================================================
// Тесты
// =============================================================================

func TestExecute_OrderCreated_SendsEmailAndPush(t *testing.T) {
	repo := &mockRepo{}
	deliverer := &mockDeliverer{}
	uc := newUseCase(repo, deliverer)

	err := uc.Execute(context.Background(), makeEvent(domain.EventOrderCreated))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// order_created → email + push
	if len(deliverer.sent) != 2 {
		t.Fatalf("expected 2 notifications sent, got %d", len(deliverer.sent))
	}

	channels := map[domain.Channel]bool{}
	for _, n := range deliverer.sent {
		channels[n.Channel] = true
	}
	if !channels[domain.ChannelEmail] {
		t.Error("expected email notification")
	}
	if !channels[domain.ChannelPush] {
		t.Error("expected push notification")
	}
}

func TestExecute_PaymentReceived_SendsEmailOnly(t *testing.T) {
	repo := &mockRepo{}
	deliverer := &mockDeliverer{}
	uc := newUseCase(repo, deliverer)

	err := uc.Execute(context.Background(), makeEvent(domain.EventPaymentReceived))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deliverer.sent) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(deliverer.sent))
	}
	if deliverer.sent[0].Channel != domain.ChannelEmail {
		t.Errorf("expected email, got %s", deliverer.sent[0].Channel)
	}
}

func TestExecute_OrderShipped_SendsPushAndSMS(t *testing.T) {
	repo := &mockRepo{}
	deliverer := &mockDeliverer{}
	uc := newUseCase(repo, deliverer)

	err := uc.Execute(context.Background(), makeEvent(domain.EventOrderShipped))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	channels := map[domain.Channel]bool{}
	for _, n := range deliverer.sent {
		channels[n.Channel] = true
	}
	if !channels[domain.ChannelPush] || !channels[domain.ChannelSMS] {
		t.Errorf("expected push+sms, got %v", channels)
	}
}

func TestExecute_DuplicateEvent_SkipsDelivery(t *testing.T) {
	repo := &mockRepo{alreadyProcessed: true}
	deliverer := &mockDeliverer{}
	uc := newUseCase(repo, deliverer)

	err := uc.Execute(context.Background(), makeEvent(domain.EventOrderCreated))

	// Дубликат — возвращаем sentinel error
	if !errors.Is(err, application.ErrAlreadyProcessed) {
		t.Fatalf("expected ErrAlreadyProcessed, got %v", err)
	}
	// Доставки не было
	if len(deliverer.sent) != 0 {
		t.Errorf("expected no deliveries for duplicate, got %d", len(deliverer.sent))
	}
}

func TestExecute_UnknownEventType_SkipsDelivery(t *testing.T) {
	repo := &mockRepo{}
	deliverer := &mockDeliverer{}
	uc := newUseCase(repo, deliverer)

	event := makeEvent("unknown_type")

	err := uc.Execute(context.Background(), event)

	if !errors.Is(err, application.ErrUnknownEventType) {
		t.Fatalf("expected ErrUnknownEventType, got %v", err)
	}
	if len(deliverer.sent) != 0 {
		t.Errorf("expected no deliveries, got %d", len(deliverer.sent))
	}
	// repo.SaveEventTx не должен был вызваться
	if repo.savedEvent.EventID != "" {
		t.Error("SaveEventTx should not be called for unknown event type")
	}
}

func TestExecute_DBError_ReturnsErrorWithoutDelivery(t *testing.T) {
	dbErr := errors.New("connection refused")
	repo := &mockRepo{saveErr: dbErr}
	deliverer := &mockDeliverer{}
	uc := newUseCase(repo, deliverer)

	err := uc.Execute(context.Background(), makeEvent(domain.EventOrderCreated))

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected db error, got %v", err)
	}
	// При ошибке БД — доставки нет
	if len(deliverer.sent) != 0 {
		t.Errorf("expected no deliveries on db error, got %d", len(deliverer.sent))
	}
}

func TestExecute_TempDeliveryError_StatusFailedTemp(t *testing.T) {
	repo := &mockRepo{}
	deliverer := &mockDeliverer{
		results: map[domain.Channel]application.DeliveryResult{
			domain.ChannelEmail: {
				Attempts: 3,
				Err:      &domain.TempError{Msg: "timeout"},
			},
		},
	}
	uc := newUseCase(repo, deliverer)

	err := uc.Execute(context.Background(), makeEvent(domain.EventPaymentReceived))

	// Ошибка доставки не всплывает — use case считается выполненным
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.updatedStatuses) != 1 {
		t.Fatalf("expected 1 status update, got %d", len(repo.updatedStatuses))
	}
	s := repo.updatedStatuses[0]
	if s.status != domain.NotificationStatusFailedTemp {
		t.Errorf("expected failed_temp, got %s", s.status)
	}
	if s.attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", s.attempts)
	}
}

func TestExecute_PermDeliveryError_StatusFailedPerm(t *testing.T) {
	repo := &mockRepo{}
	deliverer := &mockDeliverer{
		results: map[domain.Channel]application.DeliveryResult{
			domain.ChannelEmail: {
				Attempts: 1,
				Err:      &domain.PermError{Msg: "user not found"},
			},
		},
	}
	uc := newUseCase(repo, deliverer)

	err := uc.Execute(context.Background(), makeEvent(domain.EventPaymentReceived))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := repo.updatedStatuses[0]
	if s.status != domain.NotificationStatusFailedPerm {
		t.Errorf("expected failed_perm, got %s", s.status)
	}
}

func TestExecute_OneChannelFails_OtherChannelStillDelivered(t *testing.T) {
	// order_created → email + push
	// email падает с perm ошибкой, push должен доставиться
	repo := &mockRepo{}
	deliverer := &mockDeliverer{
		results: map[domain.Channel]application.DeliveryResult{
			domain.ChannelEmail: {Attempts: 1, Err: &domain.PermError{Msg: "invalid token"}},
			domain.ChannelPush:  {Attempts: 1}, // успех
		},
	}
	uc := newUseCase(repo, deliverer)

	err := uc.Execute(context.Background(), makeEvent(domain.EventOrderCreated))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deliverer.sent) != 2 {
		t.Fatalf("expected both channels attempted, got %d", len(deliverer.sent))
	}

	// Статусы: email=failed_perm, push=delivered
	statusMap := map[domain.Channel]domain.NotificationStatus{}
	for _, s := range repo.updatedStatuses {
		statusMap[s.channel] = s.status
	}
	if statusMap[domain.ChannelEmail] != domain.NotificationStatusFailedPerm {
		t.Errorf("email: expected failed_perm, got %s", statusMap[domain.ChannelEmail])
	}
	if statusMap[domain.ChannelPush] != domain.NotificationStatusDelivered {
		t.Errorf("push: expected delivered, got %s", statusMap[domain.ChannelPush])
	}
}
