package auth

import "errors"

var (
	// ErrTokenNotFound is returned when token is not found in request
	ErrTokenNotFound = errors.New("token not found in request")

	// ErrInvalidToken is returned when token is invalid
	ErrInvalidToken = errors.New("invalid token")

	// ErrTokenExpired is returned when token is expired
	ErrTokenExpired = errors.New("token is expired")

	// ErrTokenRevoked is returned when token is revoked
	ErrTokenRevoked = errors.New("token has been revoked")

	// ErrInsufficientPermissions is returned when user lacks required permissions
	ErrInsufficientPermissions = errors.New("insufficient permissions")
)
