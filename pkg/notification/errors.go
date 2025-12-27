package notification

import "errors"

var (
	// ErrUnsupportedNotificationType is returned when notification type is not supported
	ErrUnsupportedNotificationType = errors.New("unsupported notification type")
	// ErrInvalidRecipient is returned when recipient is invalid
	ErrInvalidRecipient = errors.New("invalid recipient")
	// ErrNotificationNotConfigured is returned when notification is not configured
	ErrNotificationNotConfigured = errors.New("notification not configured")
)
