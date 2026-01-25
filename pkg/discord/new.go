package discord

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"smap-collector/pkg/log"
)

// DiscordWebhook chứa thông tin webhook
type DiscordWebhook struct {
	ID    string
	Token string
}

// NewDiscordWebhook tạo webhook mới
func NewDiscordWebhook(id, token string) (*DiscordWebhook, error) {
	if id == "" || token == "" {
		return nil, errors.New("id and token are required")
	}

	return &DiscordWebhook{
		ID:    id,
		Token: token,
	}, nil
}

// Discord service implementation
type Discord struct {
	l       log.Logger
	webhook *DiscordWebhook
	config  Config
	client  *http.Client
}

// New tạo Discord service mới
func New(l log.Logger, webhook *DiscordWebhook, config Config) (*Discord, error) {
	if webhook == nil {
		return nil, errors.New("webhook is required")
	}

	if l == nil {
		return nil, errors.New("logger is required")
	}

	// Validate webhook URL
	url := fmt.Sprintf(webhookURL, webhook.ID, webhook.Token)
	if url == "" {
		return nil, errors.New("invalid webhook URL")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	return &Discord{
		l:       l,
		webhook: webhook,
		config:  config,
		client:  client,
	}, nil
}

// NewWithDefaultConfig tạo Discord service với config mặc định
func NewWithDefaultConfig(l log.Logger, webhookID, webhookToken string) (*Discord, error) {
	webhook, err := NewDiscordWebhook(webhookID, webhookToken)
	if err != nil {
		return nil, err
	}

	config := DefaultConfig()
	return New(l, webhook, config)
}

// GetWebhookURL trả về URL webhook
func (d *Discord) GetWebhookURL() string {
	return fmt.Sprintf(webhookURL, d.webhook.ID, d.webhook.Token)
}

// Close đóng HTTP client
func (d *Discord) Close() error {
	d.client.CloseIdleConnections()
	return nil
}
