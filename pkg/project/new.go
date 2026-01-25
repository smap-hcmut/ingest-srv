package project

import (
	"context"
	"net/http"
	"smap-collector/config"
	"smap-collector/pkg/log"
	"time"
)

const (
	dryRunCallbackEndpoint   = "/internal/dryrun/callback"
	progressCallbackEndpoint = "/internal/progress/callback"
)

type httpClient struct {
	client       *http.Client
	l            log.Logger
	cfg          config.ProjectConfig
	baseURL      string
	internalKey  string
	maxRetries   int
	initialDelay time.Duration
	maxDelay     time.Duration
}

func NewClient(cfg config.ProjectConfig, l log.Logger) *httpClient {
	return &httpClient{
		client: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
		l:            l,
		cfg:          cfg,
		baseURL:      cfg.BaseURL,
		internalKey:  cfg.InternalKey,
		maxRetries:   cfg.WebhookRetryAttempts,
		initialDelay: time.Duration(cfg.WebhookRetryDelay) * time.Second,
		maxDelay:     32 * time.Second,
	}
}

// Client defines the interface for the Project Service webhook client.
type Client interface {
	// SendDryRunCallback sends a dry-run webhook callback to the Project Service.
	SendDryRunCallback(ctx context.Context, req CallbackRequest) error

	// SendProgressCallback sends a progress webhook callback to the Project Service.
	SendProgressCallback(ctx context.Context, req ProgressCallbackRequest) error
}
