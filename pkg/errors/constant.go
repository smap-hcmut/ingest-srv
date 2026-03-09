package errors

import "net/http"

const (
	// HTTP status codes for predefined errors
	StatusUnauthorized = http.StatusUnauthorized // 401
	StatusForbidden    = http.StatusForbidden    // 403
)

const (
	// MessageUnauthorized is the default message for 401.
	MessageUnauthorized = "Unauthorized"
	// MessageForbidden is the default message for 403.
	MessageForbidden = "Forbidden"
)
