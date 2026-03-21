package microservice

import "errors"

var (
	ErrBadRequest    = errors.New("project client bad request")
	ErrUnauthorized  = errors.New("project client unauthorized")
	ErrForbidden     = errors.New("project client forbidden")
	ErrRequestFailed = errors.New("project client request failed")
)
