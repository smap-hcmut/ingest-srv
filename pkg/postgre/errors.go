package postgres

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidObjectIDs = errors.New("invalid object ids")
	ErrInvalidUUID      = fmt.Errorf("invalid UUID format")
)
