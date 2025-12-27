package notification

import (
	"context"
	"fmt"
)

type AliyunSMSConfig struct {
	AccessKeyId     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	Endpoint        string // 默认 cn-hangzhou
}

// AliyunSMS represents Aliyun SMS notification
type AliyunSMS struct {
	cfg AliyunSMSConfig
	cli AliyunSMSClient
}

// AliyunSMSClient is the interface for sending SMS (for dependency injection)
type AliyunSMSClient interface {
	Send(ctx context.Context, phone, sign, template string, params map[string]string) error
}

// NewAliyunSMS creates a new Aliyun SMS notification instance
func NewAliyunSMS(cfg AliyunSMSConfig, cli AliyunSMSClient) *AliyunSMS {
	return &AliyunSMS{cfg: cfg, cli: cli}
}

// Type returns the notification type
func (a *AliyunSMS) Type() NotificationType {
	return TypeSMS
}

// Send sends a notification via SMS
func (a *AliyunSMS) Send(ctx context.Context, req NotificationRequest) error {
	if a.cli == nil {
		return ErrNotificationNotConfigured
	}
	if len(req.To) == 0 {
		return ErrInvalidRecipient
	}

	// Extract code from extras or use content as code
	code := req.Content
	if req.Extras != nil {
		if c, ok := req.Extras["code"].(string); ok {
			code = c
		}
	}

	// Send to all recipients
	for _, phone := range req.To {
		params := map[string]string{"code": code}
		if err := a.cli.Send(ctx, phone, a.cfg.SignName, a.cfg.TemplateCode, params); err != nil {
			return fmt.Errorf("failed to send SMS to %s: %w", phone, err)
		}
	}
	return nil
}

// SendCode sends a verification code via SMS
func (a *AliyunSMS) SendCode(ctx context.Context, phone, code string) error {
	if a.cli == nil {
		return ErrNotificationNotConfigured
	}
	params := map[string]string{"code": code}
	return a.cli.Send(ctx, phone, a.cfg.SignName, a.cfg.TemplateCode, params)
}
