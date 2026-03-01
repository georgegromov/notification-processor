package domain

import "fmt"

// routes — таблица маршрутизации по каналам доставки.
// Правила:
//   order_created -> email + push
//   payment_received -> email
//   order_shipped -> push + sms
var routes = map[EventType][]Channel{
	EventOrderCreated:    {ChannelEmail, ChannelPush},
	EventPaymentReceived: {ChannelEmail},
	EventOrderShipped:    {ChannelPush, ChannelSMS},
}

// Route возвращает список каналов для данного типа события.
// Возвращает ошибку если event_type неизвестен.
func Route(eventType EventType) ([]Channel, error) {
	channels, ok := routes[eventType]
	if !ok {
		return nil, fmt.Errorf("unknown event_type: %q", eventType)
	}
	return channels, nil
}

// BuildNotifications создаёт список уведомлений для события.
func BuildNotifications(event Event) ([]Notification, error) {
	channels, err := Route(event.EventType)
	if err != nil {
		return nil, err
	}

	notifications := make([]Notification, 0, len(channels))
	for _, ch := range channels {
		notifications = append(notifications, Notification{
			EventID:   event.EventID,
			UserID:    event.UserID,
			Channel:   ch,
			EventType: event.EventType,
		})
	}

	return notifications, nil
}
