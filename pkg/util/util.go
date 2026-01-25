package util

func ToPointer[T any](v T) *T {
	return &v
}

func DerefSlice[T any](ptrs []*T) []T {
	result := make([]T, 0, len(ptrs))
	for _, p := range ptrs {
		if p != nil {
			result = append(result, *p)
		}
	}
	return result
}

func RemoveDuplicates(input []string) []string {
	if len(input) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(input))
	result := make([]string, 0, len(input))

	for _, item := range input {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
