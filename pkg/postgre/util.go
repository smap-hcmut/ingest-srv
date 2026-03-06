package postgres

import (
	"fmt"

	"github.com/google/uuid"
)

// IsUUID validates if the given string is a valid UUID.
// Returns an error if the string is not a valid UUID.
func IsUUID(u string) error {
	if u == "" {
		return fmt.Errorf("%w: UUID cannot be empty", ErrInvalidUUID)
	}

	_, err := uuid.Parse(u)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidUUID, err)
	}

	return nil
}

// IsValidUUID checks if the given string is a valid UUID.
// Returns true if valid, false otherwise.
func IsValidUUID(u string) bool {
	return IsUUID(u) == nil
}

// NewUUID generates a new UUID string.
func NewUUID() string {
	return uuid.New().String()
}

// ValidateUUIDs validates a slice of UUID strings.
// Returns an error if any UUID in the slice is invalid.
func ValidateUUIDs(ids []string) error {
	for i, id := range ids {
		if err := IsUUID(id); err != nil {
			return fmt.Errorf("invalid UUID at index %d: %w", i, err)
		}
	}
	return nil
}

// ConvertToInterface converts a slice of strings to a slice of interfaces.
// This is useful for SQLBoiler's WhereIn queries.
func ConvertToInterface(slice []string) []interface{} {
	interfaces := make([]interface{}, len(slice))
	for i, v := range slice {
		interfaces[i] = v
	}
	return interfaces
}
