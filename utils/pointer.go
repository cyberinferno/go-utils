package utils

// Pointer returns a pointer to the given value.
//
// Parameters:
//   - value: The value to convert to a pointer
//
// Returns:
//   - A pointer to the given value
func Pointer[T any](value T) *T {
	return &value
}
