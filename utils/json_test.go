package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsJsonString(t *testing.T) {
	t.Run("valid JSON object returns true", func(t *testing.T) {
		assert.True(t, IsJsonString(`{}`))
		assert.True(t, IsJsonString(`{"a":1}`))
		assert.True(t, IsJsonString(`{"key": "value", "n": 42}`))
	})

	t.Run("invalid JSON returns false", func(t *testing.T) {
		assert.False(t, IsJsonString(``))
		assert.False(t, IsJsonString(`not json`))
		assert.False(t, IsJsonString(`{`))
		assert.False(t, IsJsonString(`{]`))
	})

	t.Run("JSON array returns false", func(t *testing.T) {
		// Current implementation unmarshals into map[string]interface{}, so array fails
		assert.False(t, IsJsonString(`[1,2,3]`))
	})

	t.Run("JSON primitive returns false", func(t *testing.T) {
		assert.False(t, IsJsonString(`"hello"`))
		assert.False(t, IsJsonString(`123`))
		assert.False(t, IsJsonString(`true`))
	})
}
