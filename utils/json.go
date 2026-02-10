package utils

import "encoding/json"

// IsJsonString reports whether the string is valid JSON (object form).
// It attempts to unmarshal the string into a map[string]interface{} and
// returns true only if unmarshaling succeeds.
//
// Parameters:
//   - s: The string to validate
//
// Returns:
//   - true if s is valid JSON representing an object, false otherwise
func IsJsonString(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}
