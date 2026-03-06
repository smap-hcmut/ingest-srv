package discord

import (
	"net/http"
	"time"

	"ingest-srv/pkg/log"
)

// Config contains configuration for Discord service.
type Config struct {
	Timeout          time.Duration
	RetryCount       int
	RetryDelay       time.Duration
	DefaultUsername  string
	DefaultAvatarURL string
}

// webhookInfo holds parsed webhook id and token (internal use).
type webhookInfo struct {
	id    string
	token string
}

// discordImpl implements IDiscord.
type discordImpl struct {
	l       log.Logger
	webhook *webhookInfo
	config  Config
	client  *http.Client
}

// MessageType defines different types of messages.
type MessageType string

const (
	MessageTypeInfo    MessageType = "info"
	MessageTypeSuccess MessageType = "success"
	MessageTypeWarning MessageType = "warning"
	MessageTypeError   MessageType = "error"
)

// MessageLevel defines the priority level of a message.
type MessageLevel int

const (
	LevelLow MessageLevel = iota
	LevelNormal
	LevelHigh
	LevelUrgent
)

// EmbedField represents a field in a Discord embed.
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// EmbedFooter represents the footer of a Discord embed.
type EmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

// EmbedAuthor represents the author of a Discord embed.
type EmbedAuthor struct {
	Name    string `json:"name"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// Embed represents a Discord embed message.
type Embed struct {
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	URL         string          `json:"url,omitempty"`
	Color       int             `json:"color,omitempty"`
	Timestamp   string          `json:"timestamp,omitempty"`
	Footer      *EmbedFooter    `json:"footer,omitempty"`
	Author      *EmbedAuthor    `json:"author,omitempty"`
	Fields      []EmbedField    `json:"fields,omitempty"`
	Thumbnail   *EmbedThumbnail `json:"thumbnail,omitempty"`
	Image       *EmbedImage     `json:"image,omitempty"`
}

// EmbedThumbnail represents the thumbnail of an embed.
type EmbedThumbnail struct {
	URL string `json:"url"`
}

// EmbedImage represents an image in an embed.
type EmbedImage struct {
	URL string `json:"url"`
}

// WebhookPayload represents the payload sent to Discord webhook.
type WebhookPayload struct {
	Content   string  `json:"content,omitempty"`
	Username  string  `json:"username,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
	Embeds    []Embed `json:"embeds,omitempty"`
}

// MessageOptions contains options for creating a message.
type MessageOptions struct {
	Type        MessageType
	Level       MessageLevel
	Title       string
	Description string
	Fields      []EmbedField
	Footer      *EmbedFooter
	Author      *EmbedAuthor
	Thumbnail   *EmbedThumbnail
	Image       *EmbedImage
	Username    string
	AvatarURL   string
	Timestamp   time.Time
}
