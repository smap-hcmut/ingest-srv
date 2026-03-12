package scope

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"ingest-srv/internal/model"

	"github.com/golang-jwt/jwt"
)

// Verify verifies the JWT token and returns the payload if valid.
func (m *implManager) Verify(token string) (Payload, error) {
	if token == "" {
		return Payload{}, fmt.Errorf("%w: token is empty", ErrInvalidToken)
	}
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method: %v", ErrInvalidToken, t.Header["alg"])
		}
		return []byte(m.secretKey), nil
	}
	jwtToken, err := jwt.Parse(token, keyFunc)
	if err != nil {
		return Payload{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !jwtToken.Valid {
		return Payload{}, fmt.Errorf("%w: token is not valid", ErrInvalidToken)
	}
	mapClaims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return Payload{}, fmt.Errorf("%w: failed to parse claims", ErrInvalidToken)
	}
	return payloadFromMapClaims(mapClaims), nil
}

// CreateToken creates a new JWT token with the given payload.
func (m *implManager) CreateToken(payload Payload) (string, error) {
	now := time.Now()
	payload.StandardClaims = jwt.StandardClaims{
		ExpiresAt: now.Add(TokenExpirationDuration).Unix(),
		Id:        fmt.Sprintf("%d", now.UnixNano()),
		NotBefore: now.Unix(),
		IssuedAt:  now.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	return token.SignedString([]byte(m.secretKey))
}

// NewScope builds model.Scope from Payload.
func NewScope(payload Payload) model.Scope {
	userID := payload.UserID
	if userID == "" {
		userID = payload.Subject
	}
	return model.Scope{
		UserID:   userID,
		Username: payload.Username,
		Role:     payload.Role,
	}
}

// CreateScopeHeader encodes scope as base64 JSON header value.
func CreateScopeHeader(scope model.Scope) (string, error) {
	jsonData, err := json.Marshal(scope)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(jsonData), nil
}

// ParseScopeHeader decodes scope from base64 JSON header value.
func ParseScopeHeader(scopeHeader string) (model.Scope, error) {
	jsonData, err := base64.StdEncoding.DecodeString(scopeHeader)
	if err != nil {
		return model.Scope{}, err
	}
	var scope model.Scope
	if err := json.Unmarshal(jsonData, &scope); err != nil {
		return model.Scope{}, err
	}
	return scope, nil
}

// VerifyScope parses the scope header and returns the scope (no extra validation).
func (m *implManager) VerifyScope(scopeHeader string) (model.Scope, error) {
	return ParseScopeHeader(scopeHeader)
}

// SetPayloadToContext attaches Payload to context.
func SetPayloadToContext(ctx context.Context, payload Payload) context.Context {
	return context.WithValue(ctx, PayloadCtxKey{}, payload)
}

// GetPayloadFromContext returns Payload from context.
func GetPayloadFromContext(ctx context.Context) (Payload, bool) {
	payload, ok := ctx.Value(PayloadCtxKey{}).(Payload)
	return payload, ok
}

// GetUserIDFromContext returns subject/user ID from context.
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	payload, ok := GetPayloadFromContext(ctx)
	if !ok {
		return "", false
	}
	return payload.UserID, true
}

// GetUsernameFromContext returns username from context.
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	payload, ok := GetPayloadFromContext(ctx)
	if !ok {
		return "", false
	}
	return payload.Username, true
}

// SetScopeToContext attaches model.Scope to context.
func SetScopeToContext(ctx context.Context, scope model.Scope) context.Context {
	return context.WithValue(ctx, ScopeCtxKey{}, scope)
}

// GetScopeFromContext returns model.Scope from context.
func GetScopeFromContext(ctx context.Context) model.Scope {
	scope, ok := ctx.Value(ScopeCtxKey{}).(model.Scope)
	if !ok {
		return model.Scope{}
	}
	return scope
}

func payloadFromMapClaims(claims jwt.MapClaims) Payload {
	payload := Payload{
		UserID:   getStringClaim(claims, "sub"),
		Username: firstNonEmptyClaim(claims, "username", "email"),
		Role:     getStringClaim(claims, "role"),
		Type:     getStringClaim(claims, "type"),
		Refresh:  getBoolClaim(claims, "refresh"),
	}
	payload.StandardClaims = jwt.StandardClaims{
		Audience:  getAudienceClaim(claims),
		ExpiresAt: getInt64Claim(claims, "exp"),
		Id:        firstNonEmptyClaim(claims, "jti", "id"),
		IssuedAt:  getInt64Claim(claims, "iat"),
		Issuer:    getStringClaim(claims, "iss"),
		NotBefore: getInt64Claim(claims, "nbf"),
		Subject:   getStringClaim(claims, "sub"),
	}
	return payload
}

func getStringClaim(claims jwt.MapClaims, key string) string {
	value, ok := claims[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprint(v)
	}
}

func firstNonEmptyClaim(claims jwt.MapClaims, keys ...string) string {
	for _, key := range keys {
		value := getStringClaim(claims, key)
		if value != "" {
			return value
		}
	}
	return ""
}

func getInt64Claim(claims jwt.MapClaims, key string) int64 {
	value, ok := claims[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return n
		}
	}
	return 0
}

func getBoolClaim(claims jwt.MapClaims, key string) bool {
	value, ok := claims[key]
	if !ok || value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		parsed, err := strconv.ParseBool(v)
		return err == nil && parsed
	default:
		return false
	}
}

func getAudienceClaim(claims jwt.MapClaims) string {
	value, ok := claims["aud"]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				return s
			}
		}
	case []string:
		for _, item := range v {
			if item != "" {
				return item
			}
		}
	}
	return ""
}
