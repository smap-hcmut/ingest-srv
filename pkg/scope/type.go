package scope

import "github.com/golang-jwt/jwt"

// Payload represents the JWT token claims.
type Payload struct {
	jwt.StandardClaims
	UserID   string `json:"sub"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Type     string `json:"type"`
	Refresh  bool   `json:"refresh"`
}

// implManager implements Manager.
type implManager struct {
	secretKey string
}

// Context key types for payload and scope.
type (
	PayloadCtxKey       struct{}
	ScopeCtxKey         struct{}
	ThirdPartyScopeKey  struct{}
	SessionUserCtxKey   struct{}
)
