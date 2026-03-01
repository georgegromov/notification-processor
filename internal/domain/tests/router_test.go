// Тесты маршрутизации по каналам доставки.
// TestRoute — известные event_type дают список каналов, неизвестный тип ошибка.
// TestBuildNotifications — по событию список уведомлений, EventID и UserID в каждом.
package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"notification-processor/internal/domain"
)

func TestRoute(t *testing.T) {
	tests := []struct {
		eventType        domain.EventType
		expectedChannels []domain.Channel
		expectError      bool
	}{
		{
			eventType:        domain.EventOrderCreated,
			expectedChannels: []domain.Channel{domain.ChannelEmail, domain.ChannelPush},
		},
		{
			eventType:        domain.EventPaymentReceived,
			expectedChannels: []domain.Channel{domain.ChannelEmail},
		},
		{
			eventType:        domain.EventOrderShipped,
			expectedChannels: []domain.Channel{domain.ChannelPush, domain.ChannelSMS},
		},
		{
			eventType:   "unknown_event",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			channels, err := domain.Route(tt.eventType)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(channels) != len(tt.expectedChannels) {
				t.Fatalf("expected %d channels, got %d", len(tt.expectedChannels), len(channels))
			}
			for i, ch := range channels {
				if ch != tt.expectedChannels[i] {
					t.Errorf("channel[%d]: expected %q, got %q", i, tt.expectedChannels[i], ch)
				}
			}
		})
	}
}

func TestBuildNotifications(t *testing.T) {
	event := domain.NewEvent(domain.EventID(uuid.New().String()), 42, domain.EventOrderCreated, time.Now(), nil)

	notifications, err := domain.BuildNotifications(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifications) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(notifications))
	}

	// Проверяем что EventID и UserID прокинуты в каждое уведомление
	for _, n := range notifications {
		if n.EventID != event.EventID {
			t.Errorf("expected EventID %q, got %q", event.EventID, n.EventID)
		}
		if n.UserID != event.UserID {
			t.Errorf("expected UserID %d, got %d", event.UserID, n.UserID)
		}
	}
}
