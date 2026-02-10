package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBoolToYesNo(t *testing.T) {
	t.Run("true returns Yes", func(t *testing.T) {
		assert.Equal(t, "Yes", BoolToYesNo(true))
	})

	t.Run("false returns No", func(t *testing.T) {
		assert.Equal(t, "No", BoolToYesNo(false))
	})
}
