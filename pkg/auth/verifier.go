package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Verifier verifies JWT tokens using public keys from JWKS endpoint
type Verifier struct {
	jwksEndpoint string
	issuer       string
	audience     []string
	cacheTTL     time.Duration

	// Public key cache
	publicKeys map[string]*rsa.PublicKey
	keysMutex  sync.RWMutex
	lastFetch  time.Time

	// HTTP client for fetching JWKS
	httpClient *http.Client
}

// VerifierConfig holds configuration for JWT verifier
type VerifierConfig struct {
	JWKSEndpoint string
	Issuer       string
	Audience     []string
	CacheTTL     time.Duration
}

// JWKS represents JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// NewVerifier creates a new JWT verifier
func NewVerifier(cfg VerifierConfig) (*Verifier, error) {
	if cfg.JWKSEndpoint == "" {
		return nil, fmt.Errorf("JWKS endpoint is required")
	}
	if cfg.Issuer == "" {
		return nil, fmt.Errorf("issuer is required")
	}
	if len(cfg.Audience) == 0 {
		return nil, fmt.Errorf("audience is required")
	}
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 1 * time.Hour
	}

	v := &Verifier{
		jwksEndpoint: cfg.JWKSEndpoint,
		issuer:       cfg.Issuer,
		audience:     cfg.Audience,
		cacheTTL:     cfg.CacheTTL,
		publicKeys:   make(map[string]*rsa.PublicKey),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Fetch public keys on initialization
	if err := v.fetchPublicKeys(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to fetch initial public keys: %w", err)
	}

	// Start background refresh
	go v.backgroundRefresh()

	return v, nil
}

// VerifyToken verifies a JWT token and returns claims
func (v *Verifier) VerifyToken(tokenString string) (*Claims, error) {
	// Parse token without verification first to get kid
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get kid from header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid not found in token header")
		}

		// Get public key
		publicKey, err := v.getPublicKey(kid)
		if err != nil {
			return nil, err
		}

		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	// Extract claims
	claims, err := v.extractClaims(token)
	if err != nil {
		return nil, fmt.Errorf("failed to extract claims: %w", err)
	}

	// Validate claims
	if err := v.validateClaims(claims); err != nil {
		return nil, fmt.Errorf("invalid claims: %w", err)
	}

	return claims, nil
}

// extractClaims extracts claims from JWT token
func (v *Verifier) extractClaims(token *jwt.Token) (*Claims, error) {
	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	claims := &Claims{}

	// Extract standard claims
	if sub, ok := mapClaims["sub"].(string); ok {
		claims.UserID = sub
	}
	if email, ok := mapClaims["email"].(string); ok {
		claims.Email = email
	}
	if role, ok := mapClaims["role"].(string); ok {
		claims.Role = role
	}
	if jti, ok := mapClaims["jti"].(string); ok {
		claims.JTI = jti
	}
	if iss, ok := mapClaims["iss"].(string); ok {
		claims.Issuer = iss
	}
	if iat, ok := mapClaims["iat"].(float64); ok {
		claims.IssuedAt = int64(iat)
	}
	if exp, ok := mapClaims["exp"].(float64); ok {
		claims.ExpiresAt = int64(exp)
	}

	// Extract audience (can be string or array)
	if aud, ok := mapClaims["aud"].([]interface{}); ok {
		for _, a := range aud {
			if audStr, ok := a.(string); ok {
				claims.Audience = append(claims.Audience, audStr)
			}
		}
	} else if aud, ok := mapClaims["aud"].(string); ok {
		claims.Audience = []string{aud}
	}

	// Extract groups (array of strings)
	if groups, ok := mapClaims["groups"].([]interface{}); ok {
		for _, g := range groups {
			if groupStr, ok := g.(string); ok {
				claims.Groups = append(claims.Groups, groupStr)
			}
		}
	}

	return claims, nil
}

// validateClaims validates JWT claims
func (v *Verifier) validateClaims(claims *Claims) error {
	// Check expiration
	if claims.IsExpired() {
		return fmt.Errorf("token is expired")
	}

	// Check issuer
	if claims.Issuer != v.issuer {
		return fmt.Errorf("invalid issuer: expected %s, got %s", v.issuer, claims.Issuer)
	}

	// Check audience
	validAudience := false
	for _, aud := range claims.Audience {
		for _, expectedAud := range v.audience {
			if aud == expectedAud {
				validAudience = true
				break
			}
		}
		if validAudience {
			break
		}
	}
	if !validAudience {
		return fmt.Errorf("invalid audience")
	}

	return nil
}

// getPublicKey retrieves public key by kid
func (v *Verifier) getPublicKey(kid string) (*rsa.PublicKey, error) {
	v.keysMutex.RLock()

	// Check if cache is expired
	if time.Since(v.lastFetch) > v.cacheTTL {
		v.keysMutex.RUnlock()
		// Refresh keys
		if err := v.fetchPublicKeys(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to refresh public keys: %w", err)
		}
		v.keysMutex.RLock()
	}

	publicKey, ok := v.publicKeys[kid]
	v.keysMutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("public key not found for kid: %s", kid)
	}

	return publicKey, nil
}

// fetchPublicKeys fetches public keys from JWKS endpoint
func (v *Verifier) fetchPublicKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", v.jwksEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var jwks JWKS
	if err := json.Unmarshal(body, &jwks); err != nil {
		return fmt.Errorf("failed to parse JWKS: %w", err)
	}

	// Parse and store public keys
	newKeys := make(map[string]*rsa.PublicKey)
	for _, key := range jwks.Keys {
		if key.Kty != "RSA" {
			continue
		}

		publicKey, err := parseRSAPublicKey(key.N, key.E)
		if err != nil {
			return fmt.Errorf("failed to parse public key for kid %s: %w", key.Kid, err)
		}

		newKeys[key.Kid] = publicKey
	}

	// Update cache
	v.keysMutex.Lock()
	v.publicKeys = newKeys
	v.lastFetch = time.Now()
	v.keysMutex.Unlock()

	return nil
}

// backgroundRefresh periodically refreshes public keys
func (v *Verifier) backgroundRefresh() {
	ticker := time.NewTicker(v.cacheTTL / 2) // Refresh at half TTL
	defer ticker.Stop()

	for range ticker.C {
		if err := v.fetchPublicKeys(context.Background()); err != nil {
			// Log error but continue
			fmt.Printf("Failed to refresh public keys: %v\n", err)
		}
	}
}

// parseRSAPublicKey parses RSA public key from JWK n and e values
func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	// Decode base64url encoded n and e
	n, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode n: %w", err)
	}

	e, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode e: %w", err)
	}

	// Convert to big.Int
	nInt := new(big.Int)
	nInt.SetBytes(n)

	eInt := 0
	for _, b := range e {
		eInt = eInt*256 + int(b)
	}

	return &rsa.PublicKey{
		N: nInt,
		E: eInt,
	}, nil
}
