package paginator

func ConvertSlicePtrToVal[T any](src []*T) []T {
	result := make([]T, 0, len(src))
	for _, s := range src {
		if s != nil {
			result = append(result, *s)
		}
	}
	return result
}
