package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertGMTtoIST(t *testing.T) {
	t.Run("valid GMT datetime converts to IST", func(t *testing.T) {
		// 2024-01-15 12:00:00 GMT -> same time + 5:30 = 17:30 IST
		got, err := ConvertGMTtoIST("2024-01-15 12:00:00")
		require.NoError(t, err)
		assert.Equal(t, "2024-01-15 17:30:00", got)
	})

	t.Run("midnight GMT", func(t *testing.T) {
		got, err := ConvertGMTtoIST("2024-06-01 00:00:00")
		require.NoError(t, err)
		assert.Equal(t, "2024-06-01 05:30:00", got)
	})

	t.Run("invalid layout returns error", func(t *testing.T) {
		_, err := ConvertGMTtoIST("2024-01-15T12:00:00Z")
		assert.Error(t, err)
	})

	t.Run("invalid time returns error", func(t *testing.T) {
		_, err := ConvertGMTtoIST("not-a-date")
		assert.Error(t, err)
	})
}

func TestConvertUTCtoIST(t *testing.T) {
	t.Run("valid UTC datetime converts to IST", func(t *testing.T) {
		// 2024-01-15 12:00:00 UTC -> 17:30 IST
		got, err := ConvertUTCtoIST("2024-01-15T12:00:00Z")
		require.NoError(t, err)
		assert.Equal(t, "2024-01-15 17:30:00", got)
	})

	t.Run("midnight UTC", func(t *testing.T) {
		got, err := ConvertUTCtoIST("2024-06-01T00:00:00Z")
		require.NoError(t, err)
		assert.Equal(t, "2024-06-01 05:30:00", got)
	})

	t.Run("invalid layout returns error", func(t *testing.T) {
		_, err := ConvertUTCtoIST("2024-01-15 12:00:00")
		assert.Error(t, err)
	})

	t.Run("invalid time returns error", func(t *testing.T) {
		_, err := ConvertUTCtoIST("not-a-date")
		assert.Error(t, err)
	})
}
