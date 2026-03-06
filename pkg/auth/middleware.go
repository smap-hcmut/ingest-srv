package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Middleware handles JWT authentication
type Middleware struct {
	verifier       *Verifier
	blacklistRedis BlacklistChecker
	cookieName     string
}

// BlacklistChecker interface for checking if token is blacklisted
type BlacklistChecker interface {
	Exists(ctx context.Context, key string) (bool, error)
}

// MiddlewareConfig holds configuration for middleware
type MiddlewareConfig struct {
	Verifier       *Verifier
	BlacklistRedis BlacklistChecker // Optional
	CookieName     string
}

// NewMiddleware creates a new authentication middleware
func NewMiddleware(cfg MiddlewareConfig) *Middleware {
	if cfg.CookieName == "" {
		cfg.CookieName = "smap_auth_token"
	}

	return &Middleware{
		verifier:       cfg.Verifier,
		blacklistRedis: cfg.BlacklistRedis,
		cookieName:     cfg.CookieName,
	}
}

// Authenticate is a Gin middleware that verifies JWT tokens
func (m *Middleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from request
		tokenString, err := m.extractToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "MISSING_TOKEN",
					"message": "Authentication token is required",
				},
			})
			c.Abort()
			return
		}

		// Verify token
		claims, err := m.verifier.VerifyToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "INVALID_TOKEN",
					"message": "Invalid or expired authentication token",
					"details": err.Error(),
				},
			})
			c.Abort()
			return
		}

		// Check if token is blacklisted
		if m.blacklistRedis != nil {
			blacklisted, err := m.isBlacklisted(c.Request.Context(), claims.JTI)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    "BLACKLIST_CHECK_FAILED",
						"message": "Failed to verify token status",
					},
				})
				c.Abort()
				return
			}

			if blacklisted {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": gin.H{
						"code":    "TOKEN_REVOKED",
						"message": "This token has been revoked",
					},
				})
				c.Abort()
				return
			}
		}

		// Inject claims into context
		ctx := context.WithValue(c.Request.Context(), ClaimsContextKey, claims)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// extractToken extracts JWT token from Authorization header or cookie
func (m *Middleware) extractToken(c *gin.Context) (string, error) {
	// Try Authorization header first
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		// Expected format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1], nil
		}
	}

	// Try cookie
	token, err := c.Cookie(m.cookieName)
	if err == nil && token != "" {
		return token, nil
	}

	return "", ErrTokenNotFound
}

// isBlacklisted checks if token JTI is in blacklist
func (m *Middleware) isBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := "blacklist:token:" + jti
	return m.blacklistRedis.Exists(ctx, key)
}

// RequireRole returns a middleware that requires a specific role
func (m *Middleware) RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := GetClaimsFromContext(c.Request.Context())
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authentication required",
				},
			})
			c.Abort()
			return
		}

		if !claims.HasRole(role) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "INSUFFICIENT_PERMISSIONS",
					"message": "You do not have permission to access this resource",
					"details": gin.H{
						"required_role": role,
						"your_role":     claims.Role,
					},
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyRole returns a middleware that requires any of the specified roles
func (m *Middleware) RequireAnyRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := GetClaimsFromContext(c.Request.Context())
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authentication required",
				},
			})
			c.Abort()
			return
		}

		if !claims.HasAnyRole(roles...) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "INSUFFICIENT_PERMISSIONS",
					"message": "You do not have permission to access this resource",
					"details": gin.H{
						"required_roles": roles,
						"your_role":      claims.Role,
					},
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireGroup returns a middleware that requires membership in a specific group
func (m *Middleware) RequireGroup(group string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := GetClaimsFromContext(c.Request.Context())
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authentication required",
				},
			})
			c.Abort()
			return
		}

		if !claims.HasGroup(group) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "INSUFFICIENT_PERMISSIONS",
					"message": "You do not have permission to access this resource",
					"details": gin.H{
						"required_group": group,
						"your_groups":    claims.Groups,
					},
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
