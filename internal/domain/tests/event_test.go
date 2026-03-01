// Тесты Event и валидации.
// valid event — все поля заданы, успех.
// missing event_id — пустой EventID, ошибка.
// missing user_id — UserID=0, ошибка.
// missing event_type — пустой EventType, ошибка.
// missing timestamp — нулевое время, ошибка.
package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"notification-processor/internal/domain"
)

func TestEventValidate(t *testing.T) {
	validTime := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		event       domain.Event
		expectError bool
	}{
		{
			name:  "valid event",
			event: domain.NewEvent(domain.EventID(uuid.New().String()), 1, domain.EventOrderCreated, validTime, nil),
		},
		{
			name:        "missing event_id",
			event:       domain.Event{UserID: 1, EventType: domain.EventOrderCreated, Timestamp: validTime},
			expectError: true,
		},
		{
			name:        "missing user_id",
			event:       domain.Event{EventID: domain.EventID(uuid.New().String()), EventType: domain.EventOrderCreated, Timestamp: validTime},
			expectError: true,
		},
		{
			name:        "missing event_type",
			event:       domain.Event{EventID: domain.EventID(uuid.New().String()), UserID: 1, Timestamp: validTime},
			expectError: true,
		},
		{
			name:        "missing timestamp",
			event:       domain.Event{EventID: domain.EventID(uuid.New().String()), UserID: 1, EventType: domain.EventOrderCreated},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
