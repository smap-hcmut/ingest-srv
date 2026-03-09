package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// defaultTTL is used only by GenerateToken when this pkg creates tokens (e.g. auth service).
const defaultTTL = 8 * time.Hour

// Config holds JWT manager configuration (verify only: secret for signature).
type Config struct {
	SecretKey string
}

// managerImpl implements IManager.
type managerImpl struct {
	secretKey []byte
	issuer    string
	ttl       time.Duration // only used by GenerateToken
}

// Claims represents JWT claims structure.
type Claims struct {
	Email  string   `json:"email"`
	Role   string   `json:"role"`
	Groups []string `json:"groups,omitempty"`
	jwt.RegisteredClaims
}
