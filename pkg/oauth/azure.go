package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

type AzureProvider struct {
	config *oauth2.Config
}

func NewAzureProvider(cfg Config) *AzureProvider {
	return &AzureProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURI,
			Scopes:       cfg.Scopes,
			Endpoint:     microsoft.AzureADEndpoint("common"),
		},
	}
}

func (p *AzureProvider) GetAuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	return p.config.AuthCodeURL(state, opts...)
}

func (p *AzureProvider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.config.Exchange(ctx, code)
}

func (p *AzureProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", AzureUserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("microsoft graph API returned status %d", resp.StatusCode)
	}

	var azureUser struct {
		Mail        string `json:"mail"`
		DisplayName string `json:"displayName"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&azureUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &UserInfo{
		Email:   azureUser.Mail,
		Name:    azureUser.DisplayName,
		Picture: "", // Azure doesn't provide picture in basic profile
	}, nil
}

func (p *AzureProvider) GetProviderName() string {
	return "azure"
}
