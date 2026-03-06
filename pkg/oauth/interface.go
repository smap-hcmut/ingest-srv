package oauth

import (
	"context"

	"golang.org/x/oauth2"
)

// Provider defines the interface for OAuth2 identity providers.
// Implementations are safe for concurrent use.
type Provider interface {
	// GetAuthCodeURL returns the OAuth2 authorization URL
	GetAuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string

	// ExchangeCode exchanges an authorization code for an access token
	ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error)

	// GetUserInfo retrieves user information using the access token
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error)

	// GetProviderName returns the provider name (google, azure, okta)
	GetProviderName() string
}

// NewProvider creates an identity provider based on configuration.
// Returns an error if the provider type is unsupported or configuration is invalid.
func NewProvider(cfg Config) (Provider, error) {
	return newProvider(cfg)
}
