package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WeChatWorkConfig represents WeChat Work configuration
type WeChatWorkConfig struct {
	CorpID  string // Enterprise ID
	AgentID string // Application ID
	Secret  string // Application Secret
}

// WeChatWorkNotification implements WeChat Work notification
type WeChatWorkNotification struct {
	config WeChatWorkConfig
	token  string // Cached access token
	expiry time.Time
}

// NewWeChatWorkNotification creates a new WeChat Work notification instance
func NewWeChatWorkNotification(config WeChatWorkConfig) *WeChatWorkNotification {
	return &WeChatWorkNotification{config: config}
}

// Type returns the notification type
func (w *WeChatWorkNotification) Type() NotificationType {
	return TypeWeChat
}

// Send sends a notification via WeChat Work
func (w *WeChatWorkNotification) Send(ctx context.Context, req NotificationRequest) error {
	if len(req.To) == 0 {
		return ErrInvalidRecipient
	}

	// Get access token
	token, err := w.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	// Send message
	content := req.Content
	if content == "" {
		content = req.Title
	}

	return w.sendMessage(ctx, token, req.To, content)
}

// getAccessToken gets WeChat Work access token
func (w *WeChatWorkNotification) getAccessToken(ctx context.Context) (string, error) {
	// Return cached token if still valid
	if w.token != "" && time.Now().Before(w.expiry) {
		return w.token, nil
	}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		w.config.CorpID, w.config.Secret)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("wechat work API error: %s", result.ErrMsg)
	}

	// Cache token
	w.token = result.AccessToken
	w.expiry = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second) // Refresh 60s before expiry

	return w.token, nil
}

// sendMessage sends a message via WeChat Work
func (w *WeChatWorkNotification) sendMessage(ctx context.Context, token string, tousers []string, content string) error {
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", token)

	payload := map[string]interface{}{
		"touser":  joinStrings(tousers, "|"),
		"msgtype": "text",
		"agentid": w.config.AgentID,
		"text": map[string]string{
			"content": content,
		},
	}

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
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("wechat work API error: %s", result.ErrMsg)
	}

	return nil
}

// joinStrings joins string slice with separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
