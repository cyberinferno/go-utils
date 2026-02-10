package utils

import (
	"bytes"
	"math/rand"
)

var charset = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// ReadStringFromBytes interprets the byte slice as a null-terminated string.
// It returns the string up to the first null byte (0x00), or the entire buffer
// if no null byte is present.
//
// Parameters:
//   - buffer: The byte slice to read from (e.g. a fixed-size buffer from C or binary data)
//
// Returns:
//   - The string content before the first null byte, or the whole buffer as a string
func ReadStringFromBytes(buffer []byte) string {
	nullIndex := bytes.IndexByte(buffer, 0)
	if nullIndex == -1 {
		return string(buffer)
	}

	return string(buffer[:nullIndex])
}

// GenerateRandomString creates a string of the given length consisting of
// random alphanumeric characters (a-z, A-Z, 0-9).
//
// Parameters:
//   - length: The desired length of the output string
//
// Returns:
//   - A random alphanumeric string of length characters
func GenerateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}

	return string(b)
}
