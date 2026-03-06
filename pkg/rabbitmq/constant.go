package rabbitmq

import "time"

const (
	RetryConnectionDelay   = 2 * time.Second
	RetryConnectionTimeout = 20 * time.Second
	ContentTypePlainText   = "text/plain"
	ContentTypeJSON        = "application/json"
	ExchangeTypeDirect     = "direct"
	ExchangeTypeFanout     = "fanout"
	ExchangeTypeTopic      = "topic"
)
