package domain

type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusDelivered NotificationStatus = "delivered"
	// NotificationStatusFailedTemp - исчерпаны retry
	NotificationStatusFailedTemp NotificationStatus = "failed_temp"
	// NotificationStatusFailedPerm - перманентная ошибка
	NotificationStatusFailedPerm NotificationStatus = "failed_perm"
)

// Notification — одно уведомление по одному каналу
// Одно Event порождает N Notification-ов по числу каналов:
//
//  order_created -> Notification{email} + Notification{push}
//  payment_received -> Notification{email}
//  order_shipped -> Notification{push} + Notification{sms}
type Notification struct {
	EventID   EventID
	UserID    UserID
	Channel   Channel
	EventType EventType
}
