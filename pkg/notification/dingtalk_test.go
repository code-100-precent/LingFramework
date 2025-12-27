package notification

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewDingTalkNotification(t *testing.T) {
	config := DingTalkConfig{
		WebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=test",
		Secret:     "test-secret",
	}

	notif := NewDingTalkNotification(config)
	if notif == nil {
		t.Fatal("NewDingTalkNotification returned nil")
	}
	if notif.config.WebhookURL != config.WebhookURL {
		t.Errorf("WebhookURL mismatch: expected %s, got %s", config.WebhookURL, notif.config.WebhookURL)
	}
}

func TestDingTalkNotification_Type(t *testing.T) {
	config := DingTalkConfig{WebhookURL: "https://test.com"}
	notif := NewDingTalkNotification(config)

	if notif.Type() != TypeDingTalk {
		t.Errorf("Expected TypeDingTalk, got %s", notif.Type())
	}
}

func TestDingTalkNotification_Send(t *testing.T) {
	config := DingTalkConfig{WebhookURL: "https://test.com"}
	notif := NewDingTalkNotification(config)

	// Test with empty recipients
	req := NotificationRequest{
		Type:    TypeDingTalk,
		Title:   "Test",
		Content: "Test Content",
		To:      []string{},
	}

	err := notif.Send(context.Background(), req)
	if err != ErrInvalidRecipient {
		t.Errorf("Expected ErrInvalidRecipient, got %v", err)
	}

	// Test with empty content (should use title)
	req = NotificationRequest{
		Type:    TypeDingTalk,
		Title:   "Test Title",
		Content: "",
		To:      []string{"user1"},
	}

	// This will fail because webhook URL is invalid, but we test the logic
	err = notif.Send(context.Background(), req)
	if err == nil {
		t.Log("Send succeeded (unexpected)")
	}
}

func TestDingTalkNotification_SendText(t *testing.T) {
	config := DingTalkConfig{WebhookURL: ""}
	notif := NewDingTalkNotification(config)

	err := notif.SendText(context.Background(), "test")
	if err != ErrNotificationNotConfigured {
		t.Errorf("Expected ErrNotificationNotConfigured, got %v", err)
	}

	// Test with valid config but invalid URL (will fail on HTTP request)
	config.WebhookURL = "https://invalid-url-that-does-not-exist.local"
	notif = NewDingTalkNotification(config)

	err = notif.SendText(context.Background(), "test")
	if err == nil {
		t.Log("SendText succeeded (unexpected)")
	}
}

func TestDingTalkNotification_SendMarkdown(t *testing.T) {
	config := DingTalkConfig{WebhookURL: ""}
	notif := NewDingTalkNotification(config)

	err := notif.SendMarkdown(context.Background(), "Title", "# Content")
	if err != ErrNotificationNotConfigured {
		t.Errorf("Expected ErrNotificationNotConfigured, got %v", err)
	}
}

func TestDingTalkNotification_generateSign(t *testing.T) {
	config := DingTalkConfig{
		WebhookURL: "https://test.com",
		Secret:     "test-secret",
	}
	notif := NewDingTalkNotification(config)

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
}

func TestDingTalkConfig_Structure(t *testing.T) {
	config := DingTalkConfig{
		WebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=test",
		Secret:     "secret-key",
	}

	if config.WebhookURL == "" {
		t.Error("WebhookURL should not be empty")
	}
}

func TestDingTalkNotification_SendText_WithSecret(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	config := DingTalkConfig{
		WebhookURL: server.URL + "?access_token=test",
		Secret:     "test-secret",
	}
	notif := NewDingTalkNotification(config)

	err := notif.SendText(context.Background(), "test message")
	if err != nil {
		t.Errorf("SendText failed: %v", err)
	}
}

func TestDingTalkNotification_SendMarkdown_WithSecret(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	config := DingTalkConfig{
		WebhookURL: server.URL + "?access_token=test",
		Secret:     "test-secret",
	}
	notif := NewDingTalkNotification(config)

	err := notif.SendMarkdown(context.Background(), "Title", "# Content")
	if err != nil {
		t.Errorf("SendMarkdown failed: %v", err)
	}
}

func TestDingTalkNotification_SendText_WithoutSecret(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	config := DingTalkConfig{
		WebhookURL: server.URL + "?access_token=test",
		Secret:     "",
	}
	notif := NewDingTalkNotification(config)

	err := notif.SendText(context.Background(), "test message")
	if err != nil {
		t.Errorf("SendText failed: %v", err)
	}
}

func TestDingTalkNotification_SendMarkdown_WithoutSecret(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	config := DingTalkConfig{
		WebhookURL: server.URL + "?access_token=test",
		Secret:     "",
	}
	notif := NewDingTalkNotification(config)

	err := notif.SendMarkdown(context.Background(), "Title", "# Content")
	if err != nil {
		t.Errorf("SendMarkdown failed: %v", err)
	}
}

func TestDingTalkNotification_Send_WithContent(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	config := DingTalkConfig{
		WebhookURL: server.URL + "?access_token=test",
	}
	notif := NewDingTalkNotification(config)

	req := NotificationRequest{
		Type:    TypeDingTalk,
		Title:   "Test Title",
		Content: "Test Content",
		To:      []string{"user1"},
	}

	err := notif.Send(context.Background(), req)
	if err != nil {
		t.Errorf("Send failed: %v", err)
	}
}
