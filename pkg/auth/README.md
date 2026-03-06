# Auth Middleware Package

Reusable JWT authentication and authorization middleware for SMAP services.

## Features

- JWT token verification with RS256 algorithm
- Public key caching with automatic refresh
- Token blacklist checking via Redis
- Role-based authorization helpers
- Context-based user information extraction
- Gin framework integration

## Installation

```bash
go get identity-srv/pkg/auth
```

## Quick Start

### 1. Initialize Verifier

```go
import "identity-srv/pkg/auth"

// Create verifier
verifier, err := auth.NewVerifier(auth.VerifierConfig{
    JWKSEndpoint: "https://auth.example.com/.well-known/jwks.json",
    Issuer:       "smap-auth-service",
    Audience:     []string{"identity-srv"},
    CacheTTL:     1 * time.Hour,
})
if err != nil {
    log.Fatal(err)
}
```

### 2. Create Middleware

```go
// Create middleware
middleware := auth.NewMiddleware(auth.MiddlewareConfig{
    Verifier:       verifier,
    BlacklistRedis: redisClient, // Optional
    CookieName:     "smap_auth_token",
})
```

### 3. Apply to Routes

```go
r := gin.Default()

// Public routes
r.GET("/health", healthHandler)

// Protected routes
r.Use(middleware.Authenticate())
r.GET("/api/projects", listProjects)

// Role-based routes
r.POST("/api/projects", 
    middleware.RequireAnyRole("ANALYST", "ADMIN"),
    createProject,
)

r.DELETE("/api/projects/:id",
    middleware.RequireRole("ADMIN"),
    deleteProject,
)
```

## Usage Examples

### Extract User Information

```go
func myHandler(c *gin.Context) {
    // Get claims from context
    claims, ok := auth.GetClaimsFromContext(c.Request.Context())
    if !ok {
        c.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }

    // Access user information
    userID := claims.UserID
    email := claims.Email
    role := claims.Role
    groups := claims.Groups

    // Check permissions
    if claims.HasRole("ADMIN") {
        // Admin-only logic
    }

    if claims.HasAnyRole("ANALYST", "ADMIN") {
        // Analyst or Admin logic
    }

    if claims.HasGroup("marketing-team@example.com") {
        // Group-specific logic
    }
}
```

### Helper Functions

```go
// Get user ID
userID, ok := auth.GetUserIDFromContext(ctx)

// Get user role
role, ok := auth.GetUserRoleFromContext(ctx)

// Get user groups
groups, ok := auth.GetUserGroupsFromContext(ctx)
```

## Configuration

### VerifierConfig

| Field | Type | Description |
|-------|------|-------------|
| `JWKSEndpoint` | string | URL to JWKS endpoint |
| `Issuer` | string | Expected JWT issuer |
| `Audience` | []string | Expected JWT audience |
| `CacheTTL` | time.Duration | Public key cache TTL (default: 1h) |

### MiddlewareConfig

| Field | Type | Description |
|-------|------|-------------|
| `Verifier` | *Verifier | JWT verifier instance |
| `BlacklistRedis` | *redis.Client | Redis client for blacklist (optional) |
| `CookieName` | string | Cookie name for JWT (default: "smap_auth_token") |

## Error Handling

The middleware returns standard HTTP error codes:

- `401 Unauthorized`: Missing or invalid token
- `403 Forbidden`: Insufficient permissions (role check failed)

## Token Sources

The middleware checks for JWT tokens in the following order:

1. **Authorization header**: `Authorization: Bearer <token>`
2. **Cookie**: HttpOnly cookie with configured name

## Blacklist Checking

If Redis is configured, the middleware checks if the token's JTI is blacklisted:

```go
middleware := auth.NewMiddleware(auth.MiddlewareConfig{
    Verifier:       verifier,
    BlacklistRedis: redisClient, // Enable blacklist checking
    CookieName:     "smap_auth_token",
})
```

## Performance

- **JWT Verification**: < 5ms (with cached public keys)
- **Blacklist Check**: < 2ms (Redis lookup)
- **Public Key Cache**: 1 hour TTL with background refresh

## Security Best Practices

1. **Always use HTTPS** in production
2. **Set appropriate CORS** policies
3. **Enable blacklist checking** for sensitive operations
4. **Rotate JWT keys** regularly
5. **Monitor failed authentication** attempts

## Troubleshooting

### Token Verification Fails

**Symptoms:**
- 401 Unauthorized responses
- "Invalid token" errors in logs

**Solutions:**
1. Check JWKS endpoint is accessible: `curl https://auth.example.com/.well-known/jwks.json`
2. Verify issuer and audience match JWT claims
3. Ensure token is not expired (check `exp` claim)
4. Check if token is blacklisted (if Redis is configured)
5. Verify RS256 algorithm is used (not HS256)

### Public Key Cache Issues

**Symptoms:**
- Slow token verification
- "Failed to fetch public keys" errors

**Solutions:**
1. Verify JWKS endpoint returns valid JSON
2. Check network connectivity to auth service
3. Review cache TTL settings (default: 1 hour)
4. Check if background refresh is working
5. Verify firewall rules allow outbound HTTPS

### Role Authorization Fails

**Symptoms:**
- 403 Forbidden responses
- Users with correct roles denied access

**Solutions:**
1. Verify user has correct role in JWT claims: decode token at jwt.io
2. Check role name matches exactly (case-sensitive: "ADMIN" ≠ "admin")
3. Ensure JWT contains role claim (not just groups)
4. Verify middleware order: `Auth()` must come before `RequireRole()`
5. Check if role mapping is configured correctly in auth service

### Blacklist Not Working

**Symptoms:**
- Revoked tokens still accepted
- Blacklist checks failing silently

**Solutions:**
1. Verify Redis connection is configured
2. Check Redis DB number (should be 1 for blacklist)
3. Verify JTI claim exists in JWT
4. Check Redis key format: `blacklist:{jti}`
5. Ensure TTL is set correctly on blacklist entries

### Performance Issues

**Symptoms:**
- Slow API responses
- High latency on authenticated endpoints

**Solutions:**
1. Enable public key caching (should be automatic)
2. Check Redis connection pool settings
3. Monitor JWKS endpoint response time
4. Consider increasing cache TTL if keys rotate infrequently
5. Use connection pooling for Redis

### Common Errors

#### "JWKS endpoint not reachable"
```
Error: failed to fetch JWKS: Get "https://...": dial tcp: i/o timeout
```
**Fix:** Check network connectivity, firewall rules, and DNS resolution

#### "Invalid signature"
```
Error: failed to verify token: crypto/rsa: verification error
```
**Fix:** Ensure auth service and client service use same key pair

#### "Token expired"
```
Error: token is expired by 2h30m
```
**Fix:** This is expected behavior. User needs to re-authenticate.

#### "Missing required claim"
```
Error: missing required claim: role
```
**Fix:** Ensure auth service includes all required claims in JWT

## Advanced Configuration

### Custom Token Extraction

```go
// Extract token from custom header
middleware := auth.NewMiddleware(auth.MiddlewareConfig{
    Verifier:   verifier,
    CookieName: "smap_auth_token",
    // Token will be checked in Authorization header first, then cookie
})
```

### Multiple Audience Support

```go
verifier, err := auth.NewVerifier(auth.VerifierConfig{
    JWKSEndpoint: "https://auth.example.com/.well-known/jwks.json",
    Issuer:       "smap-auth-service",
    Audience:     []string{"identity-srv", "smap-mobile", "smap-web"}, // Multiple audiences
    CacheTTL:     1 * time.Hour,
})
```

### Custom Cache TTL

```go
// Shorter cache for frequently rotating keys
verifier, err := auth.NewVerifier(auth.VerifierConfig{
    JWKSEndpoint: "https://auth.example.com/.well-known/jwks.json",
    Issuer:       "smap-auth-service",
    Audience:     []string{"identity-srv"},
    CacheTTL:     15 * time.Minute, // Refresh every 15 minutes
})
```

### Conditional Authorization

```go
func myHandler(c *gin.Context) {
    claims, _ := auth.GetClaimsFromContext(c.Request.Context())
    
    // Allow ADMIN full access, ANALYST read-only
    if c.Request.Method != "GET" && !claims.HasRole("ADMIN") {
        c.JSON(403, gin.H{"error": "Admin role required for write operations"})
        return
    }
    
    // Continue with handler logic
}
```

## Integration Examples

### Project Service Integration

```go
package main

import (
    "identity-srv/pkg/auth"
    "github.com/gin-gonic/gin"
)

func main() {
    // Initialize verifier
    verifier, _ := auth.NewVerifier(auth.VerifierConfig{
        JWKSEndpoint: "https://auth.smap.com/.well-known/jwks.json",
        Issuer:       "smap-auth-service",
        Audience:     []string{"identity-srv"},
    })

    // Create middleware
    authMW := auth.NewMiddleware(auth.MiddlewareConfig{
        Verifier:       verifier,
        BlacklistRedis: redisClient,
        CookieName:     "smap_auth_token",
    })

    r := gin.Default()

    // Apply to all routes
    r.Use(authMW.Authenticate())

    // Role-based routes
    r.POST("/projects", authMW.RequireAnyRole("ANALYST", "ADMIN"), createProject)
    r.DELETE("/projects/:id", authMW.RequireRole("ADMIN"), deleteProject)

    r.Run(":8080")
}
```

### WebSocket Authentication

```go
func upgradeWebSocket(c *gin.Context) {
    // Extract JWT from query param or cookie
    token := c.Query("token")
    if token == "" {
        token, _ = c.Cookie("smap_auth_token")
    }

    // Verify token
    claims, err := verifier.Verify(token)
    if err != nil {
        c.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }

    // Check blacklist
    if blacklistRedis != nil {
        isBlacklisted, _ := checkBlacklist(claims.ID)
        if isBlacklisted {
            c.JSON(401, gin.H{"error": "Token revoked"})
            return
        }
    }

    // Upgrade connection
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        return
    }

    // Store user context in connection
    // ... handle WebSocket messages
}
```

## Examples

See `examples/` directory for complete working examples:

- `examples/basic/` - Basic authentication
- `examples/roles/` - Role-based authorization
- `examples/websocket/` - WebSocket authentication

## License

Internal use only - SMAP Platform
