package postgres

import (
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/google/uuid"
)

// IsUUID checks if the given string is a valid UUID
func IsUUID(u string) error {
	_, err := uuid.Parse(u)
	if err != nil {
		return err
	}
	return nil
}

func NewUUID() string {
	return uuid.New().String()
}

func BuildQueryWithSoftDelete() []qm.QueryMod {
	return []qm.QueryMod{
		qm.Where("deleted_at IS NULL"),
	}
}

func ConvertToInterface(slice []string) []interface{} {
	interfaces := make([]interface{}, len(slice))
	for i, v := range slice {
		interfaces[i] = v
	}
	return interfaces
}
