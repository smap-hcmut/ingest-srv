package scope

// Manager defines the interface for JWT/scope token management.
// Implementations are safe for concurrent use.
type Manager interface {
	Verify(token string) (Payload, error)
	CreateToken(payload Payload) (string, error)
}

// New creates a new scope Manager with the provided secret key.
// Panics if secretKey is empty; for safe init use NewWithConfig and check error.
func New(secretKey string) Manager {
	if secretKey == "" {
		panic("scope: secret key cannot be empty")
	}
	return &implManager{secretKey: secretKey}
}
