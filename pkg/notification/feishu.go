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

// FeishuConfig represents Feishu webhook configuration
type FeishuConfig struct {
	WebhookURL string // Feishu webhook URL
	Secret     string // Feishu webhook secret (optional)
}

// FeishuNotification implements Feishu notification
type FeishuNotification struct {
	config FeishuConfig
}

// NewFeishuNotification creates a new Feishu notification instance
func NewFeishuNotification(config FeishuConfig) *FeishuNotification {
	return &FeishuNotification{config: config}
}

// Type returns the notification type
func (f *FeishuNotification) Type() NotificationType {
	return TypeFeishu
}

// Send sends a notification via Feishu
func (f *FeishuNotification) Send(ctx context.Context, req NotificationRequest) error {
	if f.config.WebhookURL == "" {
		return ErrNotificationNotConfigured
	}

	content := req.Content
	if content == "" {
		content = req.Title
	}

	return f.SendText(ctx, content)
}

// SendText sends a text message to Feishu
func (f *FeishuNotification) SendText(ctx context.Context, content string) error {
	if f.config.WebhookURL == "" {
		return ErrNotificationNotConfigured
	}

	url := f.config.WebhookURL
	timestamp := time.Now().Unix()

	payload := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": content,
		},
	}

	// Add signature if secret is configured
	if f.config.Secret != "" {
		sign := f.generateSign(timestamp)
		payload["timestamp"] = timestamp
		payload["sign"] = sign
	}

	return f.sendRequest(ctx, url, payload)
}

// SendRichText sends a rich text message to Feishu
func (f *FeishuNotification) SendRichText(ctx context.Context, title, content string) error {
	if f.config.WebhookURL == "" {
		return ErrNotificationNotConfigured
	}

	url := f.config.WebhookURL
	timestamp := time.Now().Unix()

	payload := map[string]interface{}{
		"msg_type": "post",
		"content": map[string]interface{}{
			"post": map[string]interface{}{
				"zh_cn": map[string]interface{}{
					"title": title,
					"content": [][]map[string]string{
						{
							{"tag": "text",
								"text": content,
							},
						},
					},
				},
			},
		},
	}

	// Add signature if secret is configured
	if f.config.Secret != "" {
		sign := f.generateSign(timestamp)
		payload["timestamp"] = timestamp
		payload["sign"] = sign
	}

	return f.sendRequest(ctx, url, payload)
}

// generateSign generates signature for Feishu webhook
func (f *FeishuNotification) generateSign(timestamp int64) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, f.config.Secret)
	h := hmac.New(sha256.New, []byte(f.config.Secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// sendRequest sends HTTP request to Feishu webhook
func (f *FeishuNotification) sendRequest(ctx context.Context, url string, payload map[string]interface{}) error {
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
		return fmt.Errorf("feishu webhook returned status %d", resp.StatusCode)
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("feishu API error: %s", result.Msg)
	}

	return nil
}
