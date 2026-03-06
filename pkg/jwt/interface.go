package jwt

import (
	"ingest-srv/pkg/scope"
)

// IManager defines the interface for JWT token generation and verification.
// Implementations are safe for concurrent use.
type IManager interface {
	GenerateToken(userID, email, role string, groups []string) (string, error)
	VerifyToken(tokenString string) (*Claims, error)
	SetConfig(issuer string)
	Verify(token string) (scope.Payload, error)
	CreateToken(payload scope.Payload) (string, error)
}

// New creates a new JWT manager. Returns the interface.
func New(cfg Config) (IManager, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	return &managerImpl{
		secretKey: []byte(cfg.SecretKey),
		issuer:    "", // only used by GenerateToken; not needed for verify
		ttl:       defaultTTL,
	}, nil
}
