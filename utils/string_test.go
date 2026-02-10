package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadStringFromBytes(t *testing.T) {
	t.Run("stops at null byte", func(t *testing.T) {
		buf := []byte("hello\x00world")
		assert.Equal(t, "hello", ReadStringFromBytes(buf))
	})

	t.Run("no null returns full buffer", func(t *testing.T) {
		buf := []byte("hello")
		assert.Equal(t, "hello", ReadStringFromBytes(buf))
	})

	t.Run("null at start returns empty", func(t *testing.T) {
		buf := []byte("\x00rest")
		assert.Equal(t, "", ReadStringFromBytes(buf))
	})

	t.Run("empty buffer returns empty", func(t *testing.T) {
		assert.Equal(t, "", ReadStringFromBytes(nil))
		assert.Equal(t, "", ReadStringFromBytes([]byte{}))
	})
}

func TestGenerateRandomString(t *testing.T) {
	t.Run("length is correct", func(t *testing.T) {
		for _, n := range []int{0, 1, 10, 100} {
			got := GenerateRandomString(n)
			assert.Len(t, got, n)
		}
	})

	t.Run("only alphanumeric characters", func(t *testing.T) {
		allowed := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		allowedSet := make(map[rune]bool)
		for _, r := range allowed {
			allowedSet[r] = true
		}
		got := GenerateRandomString(200)
		for _, r := range got {
			assert.True(t, allowedSet[r], "character %q not in allowed set", r)
		}
	})

	t.Run("different calls produce different strings", func(t *testing.T) {
		// Very unlikely to get same string twice for length 32
		a := GenerateRandomString(32)
		b := GenerateRandomString(32)
		assert.NotEqual(t, a, b)
	})
}
