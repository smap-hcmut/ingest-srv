package locale

import "errors"

// ErrLocaleNotFound is returned when a requested locale is not supported.
var ErrLocaleNotFound = errors.New("locale not found")
