package notification

import (
	"context"
	"testing"
)

func TestNewNotificationManager(t *testing.T) {
	nm := NewNotificationManager()
	if nm == nil {
		t.Fatal("NewNotificationManager returned nil")
	}
	if nm.notifications == nil {
		t.Fatal("notifications map should not be nil")
	}
	if len(nm.notifications) != 0 {
		t.Fatal("notifications map should be empty initially")
	}
}

func TestNotificationManager_Register(t *testing.T) {
	nm := NewNotificationManager()

	// Create a mock notification
	mockNotif := &mockNotification{notifType: TypeEmail}

	nm.Register(mockNotif)

	if len(nm.notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(nm.notifications))
	}

	registered, ok := nm.notifications[TypeEmail]
	if !ok {
		t.Fatal("Notification not registered")
	}
	if registered != mockNotif {
		t.Fatal("Registered notification mismatch")
	}
}

func TestNotificationManager_Send(t *testing.T) {
	nm := NewNotificationManager()

	// Test with unregistered type
	req := NotificationRequest{
		Type:    TypeEmail,
		Title:   "Test",
		Content: "Test Content",
		To:      []string{"test@example.com"},
	}

	err := nm.Send(context.Background(), req)
	if err != ErrUnsupportedNotificationType {
		t.Errorf("Expected ErrUnsupportedNotificationType, got %v", err)
	}

	// Test with registered type
	mockNotif := &mockNotification{
		notifType: TypeEmail,
		sendFunc: func(ctx context.Context, req NotificationRequest) error {
			return nil
		},
	}
	nm.Register(mockNotif)

	err = nm.Send(context.Background(), req)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func TestNotificationManager_SendMultiple(t *testing.T) {
	nm := NewNotificationManager()

	// Test with no errors
	mockNotif := &mockNotification{
		notifType: TypeEmail,
		sendFunc: func(ctx context.Context, req NotificationRequest) error {
			return nil
		},
	}
	nm.Register(mockNotif)

	reqs := []NotificationRequest{
		{Type: TypeEmail, Title: "Test 1", Content: "Content 1", To: []string{"test1@example.com"}},
		{Type: TypeEmail, Title: "Test 2", Content: "Content 2", To: []string{"test2@example.com"}},
	}

	errs := nm.SendMultiple(context.Background(), reqs)
	if errs != nil {
		t.Errorf("Expected nil errors, got %v", errs)
	}

	// Test with errors
	mockNotifWithError := &mockNotification{
		notifType: TypeEmail,
		sendFunc: func(ctx context.Context, req NotificationRequest) error {
			return ErrInvalidRecipient
		},
	}
	nm.Register(mockNotifWithError)

	errs = nm.SendMultiple(context.Background(), reqs)
	if errs == nil {
		t.Fatal("Expected errors, got nil")
	}
	if len(errs) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errs))
	}
}

// mockNotification is a mock implementation of Notification interface
type mockNotification struct {
	notifType NotificationType
	sendFunc  func(ctx context.Context, req NotificationRequest) error
}

func (m *mockNotification) Type() NotificationType {
	return m.notifType
}

func (m *mockNotification) Send(ctx context.Context, req NotificationRequest) error {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, req)
	}
	return nil
}
