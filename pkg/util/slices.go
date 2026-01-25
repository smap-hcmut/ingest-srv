package util

import "math/rand"

// Map transforms a slice of one type to another using a mapping function
func Map[T, U any](slice []T, fn func(T) U) []U {
	result := make([]U, len(slice))
	for i, item := range slice {
		result[i] = fn(item)
	}
	return result
}

// MapWithIndex transforms a slice with access to both item and index
func MapWithIndex[T, U any](slice []T, fn func(int, T) U) []U {
	result := make([]U, len(slice))
	for i, item := range slice {
		result[i] = fn(i, item)
	}
	return result
}

// Filter returns a new slice containing only elements that satisfy the predicate
func Filter[T any](slice []T, fn func(T) bool) []T {
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if fn(item) {
			result = append(result, item)
		}
	}
	return result
}

// Reduce reduces a slice to a single value using a reducer function
func Reduce[T, U any](slice []T, initial U, fn func(U, T) U) U {
	result := initial
	for _, item := range slice {
		result = fn(result, item)
	}
	return result
}

// Find returns the first element that satisfies the predicate, and a boolean indicating if found
func Find[T any](slice []T, fn func(T) bool) (T, bool) {
	for _, item := range slice {
		if fn(item) {
			return item, true
		}
	}
	var zero T
	return zero, false
}

// FindIndex returns the index of the first element that satisfies the predicate, or -1 if not found
func FindIndex[T any](slice []T, fn func(T) bool) int {
	for i, item := range slice {
		if fn(item) {
			return i
		}
	}
	return -1
}

// Contains checks if a slice contains a specific element
func Contains[T comparable](slice []T, item T) bool {
	for _, element := range slice {
		if element == item {
			return true
		}
	}
	return false
}

// Unique returns a new slice with duplicate elements removed
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]bool)
	result := make([]T, 0, len(slice))
	
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// Sort sorts a slice using a comparison function
func Sort[T any](slice []T, less func(T, T) bool) {
	// Simple bubble sort implementation
	for i := 0; i < len(slice)-1; i++ {
		for j := 0; j < len(slice)-i-1; j++ {
			if less(slice[j+1], slice[j]) {
				slice[j], slice[j+1] = slice[j+1], slice[j]
			}
		}
	}
}

// Chunk splits a slice into smaller chunks of specified size
func Chunk[T any](slice []T, size int) [][]T {
	if size <= 0 {
		return [][]T{slice}
	}
	
	var result [][]T
	for i := 0; i < len(slice); i += size {
		end := i + size
		if end > len(slice) {
			end = len(slice)
		}
		result = append(result, slice[i:end])
	}
	return result
}

// Flatten flattens a slice of slices into a single slice
func Flatten[T any](slices [][]T) []T {
	var result []T
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return result
}

// Reverse reverses the order of elements in a slice
func Reverse[T any](slice []T) []T {
	result := make([]T, len(slice))
	for i, j := 0, len(slice)-1; i < len(slice); i, j = i+1, j-1 {
		result[i] = slice[j]
	}
	return result
}

// Shuffle randomly shuffles the elements in a slice
func Shuffle[T any](slice []T) []T {
	result := make([]T, len(slice))
	copy(result, slice)
	
	for i := len(result) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		result[i], result[j] = result[j], result[i]
	}
	return result
}

// GroupBy groups elements by a key function
func GroupBy[T any, K comparable](slice []T, keyFn func(T) K) map[K][]T {
	result := make(map[K][]T)
	for _, item := range slice {
		key := keyFn(item)
		result[key] = append(result[key], item)
	}
	return result
}

// Partition splits a slice into two slices based on a predicate
func Partition[T any](slice []T, fn func(T) bool) ([]T, []T) {
	var trueSlice, falseSlice []T
	for _, item := range slice {
		if fn(item) {
			trueSlice = append(trueSlice, item)
		} else {
			falseSlice = append(falseSlice, item)
		}
	}
	return trueSlice, falseSlice
} 