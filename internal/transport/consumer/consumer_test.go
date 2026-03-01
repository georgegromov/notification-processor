// Тесты parseMessage: JSON→Event, poison pill, валидация.
// ValidJSON_ReturnsEvent — корректный JSON с UUID и полями, успех.
// InvalidJSON_ReturnsError — битый JSON, ошибка json unmarshal.
// InvalidUUID_ReturnsError — event_id не UUID.
// EmptyEventID_ReturnsError — пустой event_id, invalid event_id.
// ZeroUserID_ValidationError — user_id=0, validation.
// EmptyEventType_ValidationError — пустой event_type, validation.
// ZeroTimestamp_ValidationError — нет timestamp, validation.
// EmptyPayload_Allowed — без payload допускается, Payload=nil.
package consumer

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"notification-processor/internal/domain"
)

func TestParseMessage_ValidJSON_ReturnsEvent(t *testing.T) {
	eventID := uuid.New().String()
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	payload := `{"order_id": 99, "amount": 1500}`
	raw := `{"event_id":"` + eventID + `","user_id":12345,"event_type":"order_created","timestamp":"2026-03-01T12:00:00Z","payload":` + payload + `}`

	msg := kafka.Message{Value: []byte(raw)}
	event, err := parseMessage(msg)

	require.NoError(t, err)
	assert.Equal(t, domain.EventID(eventID), event.EventID)
	assert.Equal(t, domain.UserID(12345), event.UserID)
	assert.Equal(t, domain.EventType("order_created"), event.EventType)
	assert.True(t, event.Timestamp.Equal(ts))
	assert.NotNil(t, event.Payload)
	assert.Equal(t, float64(99), event.Payload["order_id"])
	assert.Equal(t, float64(1500), event.Payload["amount"])
}

func TestParseMessage_InvalidJSON_ReturnsError(t *testing.T) {
	msg := kafka.Message{Value: []byte(`{"event_id": "broken", "user_id": }`)}
	event, err := parseMessage(msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "json unmarshal")
	assert.Zero(t, event)
}

func TestParseMessage_InvalidUUID_ReturnsError(t *testing.T) {
	raw := `{"event_id":"not-a-uuid","user_id":1,"event_type":"order_created","timestamp":"2026-03-01T12:00:00Z"}`
	msg := kafka.Message{Value: []byte(raw)}
	event, err := parseMessage(msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid event_id")
	assert.Contains(t, err.Error(), "not-a-uuid")
	assert.Zero(t, event)
}

func TestParseMessage_EmptyEventID_ReturnsError(t *testing.T) {
	// Пустой event_id отсекается сначала как invalid UUID, не доходя до Validate
	raw := `{"event_id":"","user_id":1,"event_type":"order_created","timestamp":"2026-03-01T12:00:00Z"}`
	msg := kafka.Message{Value: []byte(raw)}
	event, err := parseMessage(msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid event_id")
	assert.Zero(t, event)
}

func TestParseMessage_ZeroUserID_ValidationError(t *testing.T) {
	raw := `{"event_id":"` + uuid.New().String() + `","user_id":0,"event_type":"order_created","timestamp":"2026-03-01T12:00:00Z"}`
	msg := kafka.Message{Value: []byte(raw)}
	event, err := parseMessage(msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation")
	assert.Zero(t, event)
}

func TestParseMessage_EmptyEventType_ValidationError(t *testing.T) {
	raw := `{"event_id":"` + uuid.New().String() + `","user_id":1,"event_type":"","timestamp":"2026-03-01T12:00:00Z"}`
	msg := kafka.Message{Value: []byte(raw)}
	event, err := parseMessage(msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation")
	assert.Zero(t, event)
}

func TestParseMessage_ZeroTimestamp_ValidationError(t *testing.T) {
	// RFC3339 zero value или отсутствующее поле даёт time.Time zero
	raw := `{"event_id":"` + uuid.New().String() + `","user_id":1,"event_type":"order_created"}`
	msg := kafka.Message{Value: []byte(raw)}
	event, err := parseMessage(msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation")
	assert.Zero(t, event)
}

func TestParseMessage_EmptyPayload_Allowed(t *testing.T) {
	eventID := uuid.New().String()
	raw := `{"event_id":"` + eventID + `","user_id":1,"event_type":"payment_received","timestamp":"2026-03-01T12:00:00Z"}`
	msg := kafka.Message{Value: []byte(raw)}
	event, err := parseMessage(msg)

	require.NoError(t, err)
	assert.Equal(t, domain.EventID(eventID), event.EventID)
	assert.Nil(t, event.Payload)
}
