package model

import (
	"encoding/json"
	"time"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/types"
)

func cloneJSONRaw(src []byte) json.RawMessage {
	if len(src) == 0 {
		return nil
	}

	dst := make([]byte, len(src))
	copy(dst, src)
	return json.RawMessage(dst)
}

// jsonRawFromTypes converts non-null SQLBoiler JSON into detached raw bytes so callers can mutate safely.
func jsonRawFromTypes(src types.JSON) json.RawMessage {
	return cloneJSONRaw(src)
}

// jsonRawFromNull unwraps nullable JSON fields while preserving nil when DB value is absent.
func jsonRawFromNull(src null.JSON) json.RawMessage {
	if !src.Valid {
		return nil
	}

	return cloneJSONRaw(src.JSON)
}

// stringFromNull flattens nullable strings for API/domain structs that prefer empty string over null.String.
func stringFromNull(src null.String) string {
	if !src.Valid {
		return ""
	}

	return src.String
}

// timePtrFromNull preserves nil when timestamp is absent so optional lifecycle fields stay semantically correct.
func timePtrFromNull(src null.Time) *time.Time {
	if !src.Valid {
		return nil
	}

	t := src.Time
	return &t
}

// intPtrFromNull keeps optional integer fields distinguishable from zero values.
func intPtrFromNull(src null.Int) *int {
	if !src.Valid {
		return nil
	}

	v := src.Int
	return &v
}

// int64PtrFromNull keeps optional bigint metrics nullable in the domain layer.
func int64PtrFromNull(src null.Int64) *int64 {
	if !src.Valid {
		return nil
	}

	v := src.Int64
	return &v
}
