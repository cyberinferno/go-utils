package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRandomElement(t *testing.T) {
	t.Run("returns element from single-element slice", func(t *testing.T) {
		got := GetRandomElement([]int{42})
		assert.Equal(t, 42, got)
	})

	t.Run("returns one of the elements from slice", func(t *testing.T) {
		arr := []string{"a", "b", "c"}
		seen := make(map[string]bool)
		for _, s := range arr {
			seen[s] = true
		}
		// Run multiple times; we should get only values from the slice
		for i := 0; i < 50; i++ {
			got := GetRandomElement(arr)
			assert.True(t, seen[got], "got %q want one of %v", got, arr)
		}
	})

	t.Run("works with generic types", func(t *testing.T) {
		type point struct{ x, y int }
		arr := []point{{1, 2}, {3, 4}}
		got := GetRandomElement(arr)
		require.Contains(t, arr, got)
	})
}
