// Package utils provides common helper functions for arrays, bytes, strings,
// JSON validation, time conversion, Discord notifications, and boolean formatting.
package utils

import "math/rand"

// GetRandomElement returns a randomly chosen element from the given slice.
// The slice must be non-empty; otherwise the function panics.
//
// Parameters:
//   - arr: The slice to pick from (must have at least one element)
//
// Returns:
//   - A random element of type T from the slice
func GetRandomElement[T any](arr []T) T {
	return arr[rand.Intn(len(arr))]
}
