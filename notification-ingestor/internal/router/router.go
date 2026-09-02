package router

import (
	"fmt"

	"shared/models"
)

type Router struct {
	rules map[models.EventType][]models.NotificationChannel
}

func New() *Router {
	return &Router{
		rules: map[models.EventType][]models.NotificationChannel{
			models.EventTypeOrderCreated:    {models.NotificationChannelEmail, models.NotificationChannelPush},
			models.EventTypePaymentReceived: {models.NotificationChannelEmail},
			models.EventTypeOrderShipped:    {models.NotificationChannelPush, models.NotificationChannelSMS},
		},
	}
}

func (r *Router) ChannelsFor(eventType models.EventType) ([]models.NotificationChannel, error) {
	channels, ok := r.rules[eventType]
	if !ok {
		return nil, fmt.Errorf("unknown event_type: %s", eventType)
	}
	return channels, nil
}
