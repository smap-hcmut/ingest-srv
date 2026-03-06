package oauth

// Config holds OAuth2 provider configuration
type Config struct {
	ClientID     string   // OAuth2 client ID
	ClientSecret string   // OAuth2 client secret
	RedirectURI  string   // OAuth2 redirect URI
	Scopes       []string // OAuth2 scopes
	ProviderType string   // Provider type: "google", "azure", "okta"
	OktaDomain   string   // Okta domain (only for Okta provider)
}

// UserInfo represents normalized user information from any provider
type UserInfo struct {
	Email   string // User email address
	Name    string // User display name
	Picture string // User profile picture URL
}
