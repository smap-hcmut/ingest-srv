package discord

import "errors"

var (
	errWebhookRequired = errors.New("discord webhook URL (or id+token) is required")
)
