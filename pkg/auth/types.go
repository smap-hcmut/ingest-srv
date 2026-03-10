package auth

import (
	"context"
	"time"
)

// Claims represents JWT claims extracted from token
type Claims struct {
	UserID    string   `json:"sub"`
	Email     string   `json:"email"`
	Role      string   `json:"role"`
	Groups    []string `json:"groups"`
	JTI       string   `json:"jti"`
	Issuer    string   `json:"iss"`
	Audience  []string `json:"aud"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
}

// IsExpired checks if the token is expired
func (c *Claims) IsExpired() bool {
	return time.Now().Unix() > c.ExpiresAt
}

// HasRole checks if user has the specified role
func (c *Claims) HasRole(role string) bool {
	return c.Role == role
}

// HasAnyRole checks if user has any of the specified roles
func (c *Claims) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if c.Role == role {
			return true
		}
	}
	return false
}

// HasGroup checks if user belongs to the specified group
func (c *Claims) HasGroup(group string) bool {
	for _, g := range c.Groups {
		if g == group {
			return true
		}
	}
	return false
}

// ContextKey is the key type for storing claims in context
type ContextKey string

const (
	// ClaimsContextKey is the key for storing claims in context
	ClaimsContextKey ContextKey = "auth:claims"
)

// GetClaimsFromContext retrieves claims from context
func GetClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(*Claims)
	return claims, ok
}

// GetUserIDFromContext retrieves user ID from context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return "", false
	}
	return claims.UserID, true
}

// GetUserRoleFromContext retrieves user role from context
func GetUserRoleFromContext(ctx context.Context) (string, bool) {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return "", false
	}
	return claims.Role, true
}

// GetUserGroupsFromContext retrieves user groups from context
func GetUserGroupsFromContext(ctx context.Context) ([]string, bool) {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return nil, false
	}
	return claims.Groups, true
}
