package oauth

const (
	// ProviderGoogle represents Google OAuth2 provider
	ProviderGoogle = "google"

	// ProviderAzure represents Azure AD OAuth2 provider
	ProviderAzure = "azure"

	// ProviderOkta represents Okta OAuth2 provider
	ProviderOkta = "okta"
)

// Default scopes for each provider
var (
	// DefaultGoogleScopes are the default scopes for Google OAuth2
	DefaultGoogleScopes = []string{
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	}

	// DefaultAzureScopes are the default scopes for Azure AD OAuth2
	DefaultAzureScopes = []string{
		"openid",
		"profile",
		"email",
		"User.Read",
	}

	// DefaultOktaScopes are the default scopes for Okta OAuth2
	DefaultOktaScopes = []string{
		"openid",
		"profile",
		"email",
	}
)

// API endpoints
const (
	// GoogleUserInfoURL is the Google user info API endpoint
	GoogleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

	// AzureUserInfoURL is the Microsoft Graph API endpoint for user info
	AzureUserInfoURL = "https://graph.microsoft.com/v1.0/me"
)
