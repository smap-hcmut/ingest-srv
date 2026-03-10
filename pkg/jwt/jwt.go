package jwt

import (
	"fmt"
	"time"

	"ingest-srv/pkg/scope"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// GenerateToken generates a new JWT token with HS256.
func (m *managerImpl) GenerateToken(userID, email, role string, groups []string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.ttl)
	jti := uuid.New().String()
	claims := Claims{
		Email:  email,
		Role:   role,
		Groups: groups,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, nil
}

// VerifyToken verifies and parses a JWT token.
func (m *managerImpl) VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}
	return claims, nil
}

// SetConfig sets issuer (TTL kept as default for GenerateToken if ever used).
func (m *managerImpl) SetConfig(issuer string) {
	m.issuer = issuer
}

// Verify implements scope.Manager.
func (m *managerImpl) Verify(token string) (scope.Payload, error) {
	claims, err := m.VerifyToken(token)
	if err != nil {
		return scope.Payload{}, err
	}
	var expiresAt, issuedAt int64
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Unix()
	}
	if claims.IssuedAt != nil {
		issuedAt = claims.IssuedAt.Unix()
	}
	p := scope.Payload{
		UserID:   claims.Subject,
		Username: claims.Email,
		Role:     claims.Role,
	}
	p.Subject = claims.Subject
	p.ExpiresAt = expiresAt
	p.IssuedAt = issuedAt
	p.Id = claims.ID
	p.Issuer = claims.Issuer
	return p, nil
}

// CreateToken implements scope.Manager.
func (m *managerImpl) CreateToken(_ scope.Payload) (string, error) {
	return "", fmt.Errorf("not implemented: use GenerateToken directly")
}
