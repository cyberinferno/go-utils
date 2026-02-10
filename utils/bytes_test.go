package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeFixedLengthStringBytes(t *testing.T) {
	t.Run("short string is zero-padded", func(t *testing.T) {
		got := MakeFixedLengthStringBytes("ab", 5)
		assert.Len(t, got, 5)
		assert.Equal(t, []byte("ab"), got[:2])
		assert.Equal(t, byte(0), got[2])
		assert.Equal(t, byte(0), got[3])
		assert.Equal(t, byte(0), got[4])
	})

	t.Run("exact length unchanged", func(t *testing.T) {
		got := MakeFixedLengthStringBytes("hello", 5)
		assert.Len(t, got, 5)
		assert.Equal(t, []byte("hello"), got)
	})

	t.Run("long string is truncated", func(t *testing.T) {
		got := MakeFixedLengthStringBytes("hello world", 5)
		assert.Len(t, got, 5)
		assert.Equal(t, []byte("hello"), got)
	})

	t.Run("empty string produces zero slice", func(t *testing.T) {
		got := MakeFixedLengthStringBytes("", 3)
		assert.Len(t, got, 3)
		assert.Equal(t, []byte{0, 0, 0}, got)
	})
}

func TestJoinBytes(t *testing.T) {
	t.Run("single slice", func(t *testing.T) {
		got := JoinBytes([]byte("foo"))
		assert.Equal(t, []byte("foo"), got)
	})

	t.Run("multiple slices concatenated", func(t *testing.T) {
		got := JoinBytes([]byte("foo"), []byte("bar"), []byte("baz"))
		assert.Equal(t, []byte("foobarbaz"), got)
	})

	t.Run("empty slices", func(t *testing.T) {
		got := JoinBytes([]byte{}, []byte("a"), []byte{})
		assert.Equal(t, []byte("a"), got)
	})

	t.Run("no args returns empty", func(t *testing.T) {
		got := JoinBytes()
		assert.Empty(t, got)
	})
}
