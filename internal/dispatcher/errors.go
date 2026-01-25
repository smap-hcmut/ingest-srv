package dispatcher

import "errors"

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrUnknownRoute = errors.New("unknown platform or task type")
	ErrPublish      = errors.New("publish failed")
)
