package discord

import (
	"context"
	"fmt"
	"strings"

	"ingest-srv/pkg/log"
)

// IDiscord defines the interface for Discord webhook service.
// Implementations are safe for concurrent use.
type IDiscord interface {
	SendMessage(ctx context.Context, content string) error
	SendEmbed(ctx context.Context, options MessageOptions) error
	SendError(ctx context.Context, title, description string, err error) error
	SendSuccess(ctx context.Context, title, description string) error
	SendWarning(ctx context.Context, title, description string) error
	SendInfo(ctx context.Context, title, description string) error
	ReportBug(ctx context.Context, message string) error
	SendNotification(ctx context.Context, title, description string, fields map[string]string) error
	SendActivityLog(ctx context.Context, action, user, details string) error
	GetWebhookURL() string
	Close() error
}

// parseWebhookURL extracts id and token from Discord webhook URL (https://discord.com/api/webhooks/{id}/{token}).
func parseWebhookURL(webhookURL string) (id, token string, err error) {
	webhookURL = strings.TrimSpace(webhookURL)
	prefix := "https://discord.com/api/webhooks/"
	if !strings.HasPrefix(webhookURL, prefix) {
		return "", "", fmt.Errorf("discord: invalid webhook URL format")
	}
	rest := strings.TrimPrefix(webhookURL, prefix)
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("discord: webhook URL must be .../webhooks/{id}/{token}")
	}
	return parts[0], parts[1], nil
}

// New creates a new Discord service from webhook URL (chuỗi copy khi tạo webhook trên Discord).
func New(l log.Logger, webhookURL string) (IDiscord, error) {
	if webhookURL == "" {
		return nil, errWebhookRequired
	}
	id, token, err := parseWebhookURL(webhookURL)
	if err != nil {
		return nil, err
	}
	return newImpl(l, id, token)
}
