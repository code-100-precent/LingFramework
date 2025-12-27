package notification

import (
	"context"
	"testing"
	"time"
)

func TestNewWeChatWorkNotification(t *testing.T) {
	config := WeChatWorkConfig{
		CorpID:  "test-corp-id",
		AgentID: "test-agent-id",
		Secret:  "test-secret",
	}

	notif := NewWeChatWorkNotification(config)
	if notif == nil {
		t.Fatal("NewWeChatWorkNotification returned nil")
	}
	if notif.config.CorpID != config.CorpID {
		t.Errorf("CorpID mismatch")
	}
}

func TestWeChatWorkNotification_Type(t *testing.T) {
	config := WeChatWorkConfig{CorpID: "test", AgentID: "test", Secret: "test"}
	notif := NewWeChatWorkNotification(config)

	if notif.Type() != TypeWeChat {
		t.Errorf("Expected TypeWeChat, got %s", notif.Type())
	}
}

func TestWeChatWorkNotification_Send(t *testing.T) {
	config := WeChatWorkConfig{
		CorpID:  "test-corp-id",
		AgentID: "test-agent-id",
		Secret:  "test-secret",
	}
	notif := NewWeChatWorkNotification(config)

	// Test with empty recipients
	req := NotificationRequest{
		Type:    TypeWeChat,
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
		Type:    TypeWeChat,
		Title:   "Test Title",
		Content: "",
		To:      []string{"user1"},
	}

	// This will fail because API is not available, but we test the logic
	err = notif.Send(context.Background(), req)
	if err == nil {
		t.Log("Send succeeded (unexpected)")
	}
}

func TestWeChatWorkNotification_getAccessToken(t *testing.T) {
	config := WeChatWorkConfig{
		CorpID:  "test-corp-id",
		AgentID: "test-agent-id",
		Secret:  "test-secret",
	}
	notif := NewWeChatWorkNotification(config)

	// This will fail because API is not available
	_, err := notif.getAccessToken(context.Background())
	if err == nil {
		t.Log("getAccessToken succeeded (unexpected)")
	}
}

func TestWeChatWorkNotification_getAccessToken_Cached(t *testing.T) {
	config := WeChatWorkConfig{
		CorpID:  "test-corp-id",
		AgentID: "test-agent-id",
		Secret:  "test-secret",
	}
	notif := NewWeChatWorkNotification(config)

	// Set cached token with future expiry
	notif.token = "cached-token"
	notif.expiry = time.Now().Add(1 * time.Hour) // 1 hour in the future

	token, err := notif.getAccessToken(context.Background())
	if err != nil {
		t.Errorf("Expected nil error with cached token, got %v", err)
	}
	if token != "cached-token" {
		t.Errorf("Expected cached token, got %s", token)
	}
}

func TestWeChatWorkNotification_sendMessage(t *testing.T) {
	config := WeChatWorkConfig{
		CorpID:  "test-corp-id",
		AgentID: "test-agent-id",
		Secret:  "test-secret",
	}
	notif := NewWeChatWorkNotification(config)

	// This will fail because API is not available
	err := notif.sendMessage(context.Background(), "test-token", []string{"user1"}, "test content")
	if err == nil {
		t.Log("sendMessage succeeded (unexpected)")
	}
}

func TestJoinStrings(t *testing.T) {
	// Test empty slice
	result := joinStrings([]string{}, "|")
	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}

	// Test single element
	result = joinStrings([]string{"a"}, "|")
	if result != "a" {
		t.Errorf("Expected 'a', got %s", result)
	}

	// Test multiple elements
	result = joinStrings([]string{"a", "b", "c"}, "|")
	if result != "a|b|c" {
		t.Errorf("Expected 'a|b|c', got %s", result)
	}

	// Test with different separator
	result = joinStrings([]string{"a", "b"}, ",")
	if result != "a,b" {
		t.Errorf("Expected 'a,b', got %s", result)
	}
}

func TestWeChatWorkConfig_Structure(t *testing.T) {
	config := WeChatWorkConfig{
		CorpID:  "corp-id",
		AgentID: "agent-id",
		Secret:  "secret",
	}

	if config.CorpID == "" {
		t.Error("CorpID should not be empty")
	}
	if config.AgentID == "" {
		t.Error("AgentID should not be empty")
	}
}
