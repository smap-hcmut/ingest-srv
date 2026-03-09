package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ingest-srv/pkg/log"
)

func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}
}

// DefaultConfig returns the default Discord config.
func DefaultConfig() Config {
	return Config{
		Timeout:          DefaultTimeout,
		RetryCount:       DefaultRetryCount,
		RetryDelay:       DefaultRetryDelay,
		DefaultUsername:  DefaultUsername,
		DefaultAvatarURL: "",
	}
}

// newImpl builds discordImpl from parsed id and token (internal).
func newImpl(l log.Logger, id, token string) (IDiscord, error) {
	if id == "" || token == "" {
		return nil, errWebhookRequired
	}
	cfg := DefaultConfig()
	client := newHTTPClient(cfg.Timeout)
	return &discordImpl{
		l:       l,
		webhook: &webhookInfo{id: id, token: token},
		config:  cfg,
		client:  client,
	}, nil
}

func (d *discordImpl) GetWebhookURL() string {
	return fmt.Sprintf(webhookURLTemplate, d.webhook.id, d.webhook.token)
}

func (d *discordImpl) Close() error {
	if d.client != nil {
		d.client.CloseIdleConnections()
	}
	return nil
}

func (d *discordImpl) sendWithRetry(ctx context.Context, payload *WebhookPayload) error {
	var lastErr error
	for attempt := 0; attempt <= d.config.RetryCount; attempt++ {
		if attempt > 0 {
			if d.l != nil {
				d.l.Infof(ctx, "pkg.discord.webhook.sendWithRetry: retrying attempt %d/%d", attempt, d.config.RetryCount)
			}
			time.Sleep(d.config.RetryDelay)
		}
		err := d.sendRequest(ctx, payload)
		if err == nil {
			return nil
		}
		lastErr = err
		if d.l != nil {
			d.l.Warnf(ctx, "pkg.discord.webhook.sendWithRetry: attempt %d failed: %v", attempt+1, err)
		}
	}
	return fmt.Errorf("failed after %d attempts, last error: %w", d.config.RetryCount+1, lastErr)
}

func (d *discordImpl) sendRequest(ctx context.Context, payload *WebhookPayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	url := d.GetWebhookURL()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("discord webhook returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (d *discordImpl) validateMessageLength(content string) error {
	if len(content) > MaxMessageLength {
		return fmt.Errorf("message too long: %d characters (max: %d)", len(content), MaxMessageLength)
	}
	return nil
}

func (d *discordImpl) validateEmbedLength(embed *Embed) error {
	total := len(embed.Title) + len(embed.Description)
	for _, f := range embed.Fields {
		total += len(f.Name) + len(f.Value)
	}
	if total > MaxEmbedLength {
		return fmt.Errorf("embed too long: %d characters (max: %d)", total, MaxEmbedLength)
	}
	return nil
}

func (d *discordImpl) getColorForType(msgType MessageType) int {
	switch msgType {
	case MessageTypeInfo:
		return ColorInfo
	case MessageTypeSuccess:
		return ColorSuccess
	case MessageTypeWarning:
		return ColorWarning
	case MessageTypeError:
		return ColorError
	default:
		return ColorInfo
	}
}

func (d *discordImpl) formatTimestamp(t time.Time) string {
	return t.Format(time.RFC3339)
}

func (d *discordImpl) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func (d *discordImpl) SendMessage(ctx context.Context, content string) error {
	if err := d.validateMessageLength(content); err != nil {
		return err
	}
	payload := &WebhookPayload{
		Content:   content,
		Username:  d.config.DefaultUsername,
		AvatarURL: d.config.DefaultAvatarURL,
	}
	return d.sendWithRetry(ctx, payload)
}

func (d *discordImpl) SendEmbed(ctx context.Context, options MessageOptions) error {
	embed := &Embed{
		Title:       d.truncateString(options.Title, MaxTitleLen),
		Description: d.truncateString(options.Description, MaxDescriptionLen),
		Color:       d.getColorForType(options.Type),
		Fields:      options.Fields,
		Footer:      options.Footer,
		Author:      options.Author,
		Thumbnail:   options.Thumbnail,
		Image:       options.Image,
	}
	if !options.Timestamp.IsZero() {
		embed.Timestamp = d.formatTimestamp(options.Timestamp)
	}
	if err := d.validateEmbedLength(embed); err != nil {
		return err
	}
	payload := &WebhookPayload{
		Embeds:    []Embed{*embed},
		Username:  options.Username,
		AvatarURL: options.AvatarURL,
	}
	if payload.Username == "" {
		payload.Username = d.config.DefaultUsername
	}
	if payload.AvatarURL == "" {
		payload.AvatarURL = d.config.DefaultAvatarURL
	}
	return d.sendWithRetry(ctx, payload)
}

func (d *discordImpl) SendError(ctx context.Context, title, description string, err error) error {
	fields := []EmbedField{}
	if err != nil {
		fields = append(fields, EmbedField{
			Name:   "Error",
			Value:  d.truncateString(err.Error(), MaxFieldValueLen),
			Inline: false,
		})
	}
	return d.SendEmbed(ctx, MessageOptions{
		Type:        MessageTypeError,
		Level:       LevelHigh,
		Title:       title,
		Description: description,
		Fields:      fields,
		Timestamp:   time.Now(),
	})
}

func (d *discordImpl) SendSuccess(ctx context.Context, title, description string) error {
	return d.SendEmbed(ctx, MessageOptions{
		Type: MessageTypeSuccess, Level: LevelNormal,
		Title: title, Description: description, Timestamp: time.Now(),
	})
}

func (d *discordImpl) SendWarning(ctx context.Context, title, description string) error {
	return d.SendEmbed(ctx, MessageOptions{
		Type: MessageTypeWarning, Level: LevelNormal,
		Title: title, Description: description, Timestamp: time.Now(),
	})
}

func (d *discordImpl) SendInfo(ctx context.Context, title, description string) error {
	return d.SendEmbed(ctx, MessageOptions{
		Type: MessageTypeInfo, Level: LevelNormal,
		Title: title, Description: description, Timestamp: time.Now(),
	})
}

func (d *discordImpl) ReportBug(ctx context.Context, message string) error {
	if len(message) > ReportBugDescLen {
		message = message[:ReportBugDescLen-3] + "..."
	}
	return d.SendEmbed(ctx, MessageOptions{
		Type:        MessageTypeError,
		Level:       LevelUrgent,
		Title:       ReportBugTitle,
		Description: fmt.Sprintf("```%s```", message),
		Timestamp:   time.Now(),
	})
}

func (d *discordImpl) SendNotification(ctx context.Context, title, description string, fields map[string]string) error {
	var embedFields []EmbedField
	for name, value := range fields {
		embedFields = append(embedFields, EmbedField{
			Name:   d.truncateString(name, MaxTitleLen),
			Value:  d.truncateString(value, MaxFieldValueLen),
			Inline: true,
		})
	}
	return d.SendEmbed(ctx, MessageOptions{
		Type: MessageTypeInfo, Level: LevelNormal,
		Title: title, Description: description,
		Fields: embedFields, Timestamp: time.Now(),
	})
}

func (d *discordImpl) SendActivityLog(ctx context.Context, action, user, details string) error {
	fields := []EmbedField{
		{Name: "Action", Value: action, Inline: true},
		{Name: "User", Value: user, Inline: true},
	}
	if details != "" {
		fields = append(fields, EmbedField{Name: "Details", Value: details, Inline: false})
	}
	return d.SendEmbed(ctx, MessageOptions{
		Type:        MessageTypeInfo,
		Level:       LevelLow,
		Title:       ActivityLogTitle,
		Description: fmt.Sprintf("**%s** performed **%s**", user, action),
		Fields:      fields,
		Timestamp:   time.Now(),
	})
}
