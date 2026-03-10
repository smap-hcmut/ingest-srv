package util

// MapSlice applies a converter function to each element of a slice and returns a new slice.
// If the converter returns nil, that element is skipped.
func MapSlice[T any, R any](items []*T, converter func(*T) *R) []R {
	result := make([]R, 0, len(items))
	for _, item := range items {
		if converted := converter(item); converted != nil {
			result = append(result, *converted)
		}
	}
	return result
}

// ToInterfaceSlice converts a slice of any type to []interface{}.
// Useful for SQLBoiler WhereIn and similar operations that require []interface{}.
func ToInterfaceSlice[T any](items []T) []interface{} {
	result := make([]interface{}, len(items))
	for i, item := range items {
		result[i] = item
	}
	return result
}
