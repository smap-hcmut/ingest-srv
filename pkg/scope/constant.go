package scope

import "time"

const (
	// TokenExpirationDuration is the default JWT token expiration (1 week).
	TokenExpirationDuration = time.Hour * 24 * 7
)
