package domain

// Каналы доставки уведомлений

// Channel — тип канала доставки
type Channel string

const (
	ChannelEmail Channel = "email"
	ChannelPush  Channel = "push"
	ChannelSMS   Channel = "sms"
)
