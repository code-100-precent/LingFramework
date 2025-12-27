package notification

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewFeishuNotification(t *testing.T) {
	config := FeishuConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
		Secret:     "test-secret",
	}

	notif := NewFeishuNotification(config)
	if notif == nil {
		t.Fatal("NewFeishuNotification returned nil")
	}
	if notif.config.WebhookURL != config.WebhookURL {
		t.Errorf("WebhookURL mismatch")
	}
}

func TestFeishuNotification_Type(t *testing.T) {
	config := FeishuConfig{WebhookURL: "https://test.com"}
	notif := NewFeishuNotification(config)

	if notif.Type() != TypeFeishu {
		t.Errorf("Expected TypeFeishu, got %s", notif.Type())
	}
}

func TestFeishuNotification_Send(t *testing.T) {
	config := FeishuConfig{WebhookURL: ""}
	notif := NewFeishuNotification(config)

	req := NotificationRequest{
		Type:    TypeFeishu,
		Title:   "Test",
		Content: "Test Content",
		To:      []string{"user1"},
	}

	err := notif.Send(context.Background(), req)
	if err != ErrNotificationNotConfigured {
		t.Errorf("Expected ErrNotificationNotConfigured, got %v", err)
	}

	// Test with empty content (should use title)
	config.WebhookURL = "https://invalid-url.local"
	notif = NewFeishuNotification(config)
	req.Content = ""

	err = notif.Send(context.Background(), req)
	if err == nil {
		t.Log("Send succeeded (unexpected)")
	}
}

func TestFeishuNotification_SendText(t *testing.T) {
	config := FeishuConfig{WebhookURL: ""}
	notif := NewFeishuNotification(config)

	err := notif.SendText(context.Background(), "test")
	if err != ErrNotificationNotConfigured {
		t.Errorf("Expected ErrNotificationNotConfigured, got %v", err)
	}

	// Test with mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":0,"msg":"success"}`))
	}))
	defer server.Close()

	config = FeishuConfig{
		WebhookURL: server.URL,
		Secret:     "test-secret",
	}
	notif = NewFeishuNotification(config)

	err = notif.SendText(context.Background(), "test")
	if err != nil {
		t.Errorf("SendText failed: %v", err)
	}

	// Test without secret
	config = FeishuConfig{
		WebhookURL: server.URL,
		Secret:     "",
	}
	notif = NewFeishuNotification(config)

	err = notif.SendText(context.Background(), "test")
	if err != nil {
		t.Errorf("SendText failed: %v", err)
	}
}

func TestFeishuNotification_SendRichText(t *testing.T) {
	config := FeishuConfig{WebhookURL: ""}
	notif := NewFeishuNotification(config)

	err := notif.SendRichText(context.Background(), "Title", "Content")
	if err != ErrNotificationNotConfigured {
		t.Errorf("Expected ErrNotificationNotConfigured, got %v", err)
	}

	// Test with mock server and secret
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":0,"msg":"success"}`))
	}))
	defer server.Close()

	config = FeishuConfig{
		WebhookURL: server.URL,
		Secret:     "test-secret",
	}
	notif = NewFeishuNotification(config)

	err = notif.SendRichText(context.Background(), "Title", "Content")
	if err != nil {
		t.Errorf("SendRichText failed: %v", err)
	}

	// Test without secret
	config = FeishuConfig{
		WebhookURL: server.URL,
		Secret:     "",
	}
	notif = NewFeishuNotification(config)

	err = notif.SendRichText(context.Background(), "Title", "Content")
	if err != nil {
		t.Errorf("SendRichText failed: %v", err)
	}
}

func TestFeishuNotification_generateSign(t *testing.T) {
	config := FeishuConfig{
		WebhookURL: "https://test.com",
		Secret:     "test-secret",
	}
	notif := NewFeishuNotification(config)

	timestamp := int64(1234567890)
	sign := notif.generateSign(timestamp)

	if sign == "" {
		t.Error("generateSign returned empty string")
	}

	// Test that same input produces same output
	sign2 := notif.generateSign(timestamp)
	if sign != sign2 {
		t.Error("generateSign should produce consistent output")
	}

	// Test with different timestamp
	sign3 := notif.generateSign(timestamp + 1)
	if sign == sign3 {
		t.Error("generateSign should produce different output for different timestamp")
	}
}

func TestFeishuConfig_Structure(t *testing.T) {
	config := FeishuConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
		Secret:     "secret-key",
	}

	if config.WebhookURL == "" {
		t.Error("WebhookURL should not be empty")
	}
}
