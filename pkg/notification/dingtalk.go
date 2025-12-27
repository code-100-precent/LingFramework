package notification

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DingTalkConfig represents DingTalk webhook configuration
type DingTalkConfig struct {
	WebhookURL string // DingTalk webhook URL
	Secret     string // DingTalk webhook secret (optional)
}

// DingTalkNotification implements DingTalk notification
type DingTalkNotification struct {
	config DingTalkConfig
}

// NewDingTalkNotification creates a new DingTalk notification instance
func NewDingTalkNotification(config DingTalkConfig) *DingTalkNotification {
	return &DingTalkNotification{config: config}
}

// Type returns the notification type
func (d *DingTalkNotification) Type() NotificationType {
	return TypeDingTalk
}

// Send sends a notification via DingTalk
func (d *DingTalkNotification) Send(ctx context.Context, req NotificationRequest) error {
	if len(req.To) == 0 {
		return ErrInvalidRecipient
	}

	// Use title as content if content is empty
	content := req.Content
	if content == "" {
		content = req.Title
	}

	// Send text message by default
	return d.SendText(ctx, content)
}

// SendText sends a text message to DingTalk
func (d *DingTalkNotification) SendText(ctx context.Context, content string) error {
	if d.config.WebhookURL == "" {
		return ErrNotificationNotConfigured
	}

	url := d.config.WebhookURL
	if d.config.Secret != "" {
		// Generate signed URL
		timestamp := time.Now().UnixNano() / 1e6
		sign := d.generateSign(timestamp)
		url = fmt.Sprintf("%s&timestamp=%d&sign=%s", d.config.WebhookURL, timestamp, sign)
	}

	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
	}

	return d.sendRequest(ctx, url, payload)
}

// SendMarkdown sends a markdown message to DingTalk
func (d *DingTalkNotification) SendMarkdown(ctx context.Context, title, content string) error {
	if d.config.WebhookURL == "" {
		return ErrNotificationNotConfigured
	}

	url := d.config.WebhookURL
	if d.config.Secret != "" {
		timestamp := time.Now().UnixNano() / 1e6
		sign := d.generateSign(timestamp)
		url = fmt.Sprintf("%s&timestamp=%d&sign=%s", d.config.WebhookURL, timestamp, sign)
	}

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  content,
		},
	}

	return d.sendRequest(ctx, url, payload)
}

// generateSign generates signature for DingTalk webhook
func (d *DingTalkNotification) generateSign(timestamp int64) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, d.config.Secret)
	h := hmac.New(sha256.New, []byte(d.config.Secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// sendRequest sends HTTP request to DingTalk webhook
func (d *DingTalkNotification) sendRequest(ctx context.Context, url string, payload map[string]interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dingtalk webhook returned status %d", resp.StatusCode)
	}

	return nil
}
