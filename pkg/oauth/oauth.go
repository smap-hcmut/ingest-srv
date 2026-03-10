package oauth

import (
	"fmt"
)

// newProvider creates an identity provider based on configuration (internal)
func newProvider(cfg Config) (Provider, error) {
	switch cfg.ProviderType {
	case ProviderGoogle:
		return NewGoogleProvider(cfg), nil
	case ProviderAzure:
		return NewAzureProvider(cfg), nil
	case ProviderOkta:
		if cfg.OktaDomain == "" {
			return nil, fmt.Errorf("okta_domain is required for Okta provider")
		}
		return NewOktaProvider(cfg, cfg.OktaDomain), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s (supported: %s, %s, %s)",
			cfg.ProviderType, ProviderGoogle, ProviderAzure, ProviderOkta)
	}
}
