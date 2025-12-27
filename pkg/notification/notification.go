package notification

import (
	"context"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	TypeInternal NotificationType = "internal" // Internal notification (in-app)
	TypeEmail    NotificationType = "email"    // Email notification
	TypeSMS      NotificationType = "sms"      // SMS notification
	TypePush     NotificationType = "push"     // Push notification (JPush, etc.)
	TypeDingTalk NotificationType = "dingtalk" // DingTalk notification
	TypeWeChat   NotificationType = "wechat"   // WeChat Work notification
	TypeFeishu   NotificationType = "feishu"   // Feishu notification
)

// NotificationRequest represents a unified notification request
type NotificationRequest struct {
	Type    NotificationType       // Notification type
	Title   string                 // Notification title
	Content string                 // Notification content
	To      []string               // Recipients (email addresses, phone numbers, user IDs, etc.)
	Extras  map[string]interface{} // Extra data for specific notification types
	Context context.Context        // Context for cancellation/timeout
}

// Notification is the unified notification interface
type Notification interface {
	// Send sends a notification
	Send(ctx context.Context, req NotificationRequest) error
	// Type returns the notification type
	Type() NotificationType
}

// NotificationManager manages multiple notification channels
type NotificationManager struct {
	notifications map[NotificationType]Notification
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager() *NotificationManager {
	return &NotificationManager{
		notifications: make(map[NotificationType]Notification),
	}
}

// Register registers a notification handler
func (nm *NotificationManager) Register(notif Notification) {
	nm.notifications[notif.Type()] = notif
}

// Send sends a notification using the appropriate handler
func (nm *NotificationManager) Send(ctx context.Context, req NotificationRequest) error {
	handler, ok := nm.notifications[req.Type]
	if !ok {
		return ErrUnsupportedNotificationType
	}
	return handler.Send(ctx, req)
}

// SendMultiple sends notifications to multiple channels
func (nm *NotificationManager) SendMultiple(ctx context.Context, reqs []NotificationRequest) []error {
	var errors []error
	for _, req := range reqs {
		if err := nm.Send(ctx, req); err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return errors
	}
	return nil
}
