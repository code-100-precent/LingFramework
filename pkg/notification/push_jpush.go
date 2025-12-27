package notification

import "context"

type JPushConfig struct {
	AppKey       string
	MasterSecret string
}

type JPushClient interface {
	Push(ctx context.Context, title, content string, audience map[string]interface{}, extras map[string]interface{}) error
}

// JPush represents JPush notification
type JPush struct {
	cfg JPushConfig
	cli JPushClient
}

// NewJPush creates a new JPush notification instance
func NewJPush(cfg JPushConfig, cli JPushClient) *JPush {
	return &JPush{cfg: cfg, cli: cli}
}

// Type returns the notification type
func (j *JPush) Type() NotificationType {
	return TypePush
}

// Send sends a notification via JPush
func (j *JPush) Send(ctx context.Context, req NotificationRequest) error {
	if j.cli == nil {
		return ErrNotificationNotConfigured
	}

	// Extract alias from To field or extras
	var alias []string
	if len(req.To) > 0 {
		alias = req.To
	} else if req.Extras != nil {
		if a, ok := req.Extras["alias"].([]string); ok {
			alias = a
		}
	}

	// If no alias, send to all
	if len(alias) == 0 {
		return j.PushToAll(ctx, req.Title, req.Content, req.Extras)
	}

	return j.PushToAlias(ctx, alias, req.Title, req.Content, req.Extras)
}

// PushToAlias pushes notification to specific aliases
func (j *JPush) PushToAlias(ctx context.Context, alias []string, title, content string, extras map[string]interface{}) error {
	if j.cli == nil {
		return ErrNotificationNotConfigured
	}
	aud := map[string]interface{}{"alias": alias}
	return j.cli.Push(ctx, title, content, aud, extras)
}

// PushToAll pushes notification to all users
func (j *JPush) PushToAll(ctx context.Context, title, content string, extras map[string]interface{}) error {
	if j.cli == nil {
		return ErrNotificationNotConfigured
	}
	aud := map[string]interface{}{"all": true}
	return j.cli.Push(ctx, title, content, aud, extras)
}
