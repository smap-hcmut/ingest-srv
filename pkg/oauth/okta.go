package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

type OktaProvider struct {
	config     *oauth2.Config
	oktaDomain string
}

func NewOktaProvider(cfg Config, oktaDomain string) *OktaProvider {
	return &OktaProvider{
		oktaDomain: oktaDomain,
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURI,
			Scopes:       cfg.Scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  fmt.Sprintf("https://%s/oauth2/v1/authorize", oktaDomain),
				TokenURL: fmt.Sprintf("https://%s/oauth2/v1/token", oktaDomain),
			},
		},
	}
}

func (p *OktaProvider) GetAuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	return p.config.AuthCodeURL(state, opts...)
}

func (p *OktaProvider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.config.Exchange(ctx, code)
}

func (p *OktaProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	userInfoURL := fmt.Sprintf("https://%s/oauth2/v1/userinfo", p.oktaDomain)
	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
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
		return nil, fmt.Errorf("okta API returned status %d", resp.StatusCode)
	}

	var oktaUser struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&oktaUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &UserInfo{
		Email:   oktaUser.Email,
		Name:    oktaUser.Name,
		Picture: oktaUser.Picture,
	}, nil
}

func (p *OktaProvider) GetProviderName() string {
	return "okta"
}
