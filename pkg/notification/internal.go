package notification

import (
	"context"
	"fmt"
	"strconv"
)

// InternalNotificationAdapter adapts InternalNotificationService to Notification interface
type InternalNotificationAdapter struct {
	service *InternalNotificationService
}

// NewInternalNotificationAdapter creates a new internal notification adapter
func NewInternalNotificationAdapter(service *InternalNotificationService) *InternalNotificationAdapter {
	return &InternalNotificationAdapter{service: service}
}

// Type returns the notification type
func (i *InternalNotificationAdapter) Type() NotificationType {
	return TypeInternal
}

// Send sends an internal notification
func (i *InternalNotificationAdapter) Send(ctx context.Context, req NotificationRequest) error {
	if len(req.To) == 0 {
		return ErrInvalidRecipient
	}

	// Send to all user IDs
	for _, userIDStr := range req.To {
		userID, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid user ID: %s: %w", userIDStr, err)
		}
		if err := i.service.Send(uint(userID), req.Title, req.Content); err != nil {
			return fmt.Errorf("failed to send internal notification to user %d: %w", userID, err)
		}
	}
	return nil
}
