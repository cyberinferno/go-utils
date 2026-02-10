package utils

// MakeFixedLengthStringBytes creates a byte slice of the given length containing
// the string's bytes. If the string is shorter than length, the remainder is
// zero-padded; if longer, the string is truncated.
//
// Parameters:
//   - str: The string to convert to bytes
//   - length: The fixed length of the resulting byte slice
//
// Returns:
//   - A byte slice of length bytes with the string content (padded or truncated)
func MakeFixedLengthStringBytes(str string, length int) []byte {
	bytesMsg := make([]byte, length)
	strBytes := []byte(str)
	copy(bytesMsg, strBytes)
	return bytesMsg
}

// JoinBytes concatenates the given byte slices into a single byte slice.
//
// Parameters:
//   - s: One or more byte slices to concatenate
//
// Returns:
//   - A new byte slice containing all input slices in order
func JoinBytes(s ...[]byte) []byte {
	n := 0
	for _, v := range s {
		n += len(v)
	}

	b, i := make([]byte, n), 0
	for _, v := range s {
		i += copy(b[i:], v)
	}

	return b
}
