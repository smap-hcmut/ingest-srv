package scope

import "errors"

var (
	// ErrInvalidToken is returned when a JWT token is invalid, expired, or malformed.
	ErrInvalidToken = errors.New("invalid token")
)
