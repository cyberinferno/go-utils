package utils

// BoolToYesNo converts a boolean value to a human-readable "Yes" or "No" string.
//
// Parameters:
//   - value: The boolean to convert
//
// Returns:
//   - "Yes" if value is true, "No" if value is false
func BoolToYesNo(value bool) string {
	if value {
		return "Yes"
	}

	return "No"
}
