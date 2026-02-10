package perfmonitor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewPerformanceMonitor(t *testing.T) {
	t.Run("creates new instance with zero times", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		assert.NotNil(t, pm)
		assert.True(t, pm.startTime.IsZero())
		assert.True(t, pm.endTime.IsZero())
	})
}

func TestStart(t *testing.T) {
	t.Run("sets start time", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Start()

		assert.False(t, pm.startTime.IsZero())
		assert.True(t, pm.endTime.IsZero())
	})

	t.Run("overwrites previous start time on multiple calls", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Start()
		firstStartTime := pm.startTime
		time.Sleep(10 * time.Millisecond)
		pm.Start()
		secondStartTime := pm.startTime

		assert.True(t, secondStartTime.After(firstStartTime))
		assert.True(t, pm.endTime.IsZero())
	})
}

func TestStop(t *testing.T) {
	t.Run("sets end time after start", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Start()
		pm.Stop()

		assert.False(t, pm.startTime.IsZero())
		assert.False(t, pm.endTime.IsZero())
		assert.True(t, pm.endTime.After(pm.startTime) || pm.endTime.Equal(pm.startTime))
	})

	t.Run("does not set end time if start was not called", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Stop()

		assert.True(t, pm.startTime.IsZero())
		assert.True(t, pm.endTime.IsZero())
	})

	t.Run("overwrites previous end time on multiple calls", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Start()
		pm.Stop()
		firstEndTime := pm.endTime
		time.Sleep(10 * time.Millisecond)
		pm.Stop()
		secondEndTime := pm.endTime

		assert.True(t, secondEndTime.After(firstEndTime))
	})

	t.Run("does not set end time if start was reset", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Start()
		pm.Reset()
		pm.Stop()

		assert.True(t, pm.startTime.IsZero())
		assert.True(t, pm.endTime.IsZero())
	})
}

func TestElapsedMilliseconds(t *testing.T) {
	t.Run("returns zero if start was not called", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		elapsed := pm.ElapsedMilliseconds()

		assert.Equal(t, 0.0, elapsed)
	})

	t.Run("returns zero if stop was not called", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Start()
		elapsed := pm.ElapsedMilliseconds()

		assert.Equal(t, 0.0, elapsed)
	})

	t.Run("returns zero if both start and stop were not called", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		elapsed := pm.ElapsedMilliseconds()

		assert.Equal(t, 0.0, elapsed)
	})

	t.Run("returns elapsed time in milliseconds", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Start()
		time.Sleep(100 * time.Millisecond)
		pm.Stop()

		elapsed := pm.ElapsedMilliseconds()

		assert.Greater(t, elapsed, 90.0) // Allow some margin for timing
		assert.Less(t, elapsed, 150.0)   // Allow some margin for timing
	})

	t.Run("returns accurate elapsed time for short duration", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Start()
		time.Sleep(10 * time.Millisecond)
		pm.Stop()

		elapsed := pm.ElapsedMilliseconds()

		assert.Greater(t, elapsed, 5.0) // Allow some margin for timing
		assert.Less(t, elapsed, 50.0)   // Allow some margin for timing
	})

	t.Run("returns zero after reset", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Start()
		time.Sleep(10 * time.Millisecond)
		pm.Stop()
		pm.Reset()

		elapsed := pm.ElapsedMilliseconds()

		assert.Equal(t, 0.0, elapsed)
	})

	t.Run("returns updated elapsed time after multiple stop calls", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Start()
		time.Sleep(50 * time.Millisecond)
		pm.Stop()
		firstElapsed := pm.ElapsedMilliseconds()

		time.Sleep(50 * time.Millisecond)
		pm.Stop()
		secondElapsed := pm.ElapsedMilliseconds()

		assert.Greater(t, secondElapsed, firstElapsed)
		assert.Greater(t, secondElapsed, 90.0)
		assert.Less(t, secondElapsed, 150.0)
	})

	t.Run("returns zero if start was reset before stop", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Start()
		pm.Reset()
		pm.Stop()

		elapsed := pm.ElapsedMilliseconds()

		assert.Equal(t, 0.0, elapsed)
	})
}

func TestReset(t *testing.T) {
	t.Run("clears start and end times", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Start()
		pm.Stop()
		pm.Reset()

		assert.True(t, pm.startTime.IsZero())
		assert.True(t, pm.endTime.IsZero())
	})

	t.Run("clears only start time if stop was not called", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Start()
		pm.Reset()

		assert.True(t, pm.startTime.IsZero())
		assert.True(t, pm.endTime.IsZero())
	})

	t.Run("allows reuse after reset", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		// First measurement
		pm.Start()
		time.Sleep(50 * time.Millisecond)
		pm.Stop()
		firstElapsed := pm.ElapsedMilliseconds()

		// Reset and reuse
		pm.Reset()
		pm.Start()
		time.Sleep(50 * time.Millisecond)
		pm.Stop()
		secondElapsed := pm.ElapsedMilliseconds()

		assert.Greater(t, firstElapsed, 40.0)
		assert.Less(t, firstElapsed, 100.0)
		assert.Greater(t, secondElapsed, 40.0)
		assert.Less(t, secondElapsed, 100.0)
	})

	t.Run("multiple resets are safe", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Start()
		pm.Stop()
		pm.Reset()
		pm.Reset()
		pm.Reset()

		assert.True(t, pm.startTime.IsZero())
		assert.True(t, pm.endTime.IsZero())
	})
}

func TestPerformanceMonitor_CompleteWorkflow(t *testing.T) {
	t.Run("complete start-stop-elapsed workflow", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Start()
		time.Sleep(75 * time.Millisecond)
		pm.Stop()

		elapsed := pm.ElapsedMilliseconds()

		assert.Greater(t, elapsed, 60.0)
		assert.Less(t, elapsed, 120.0)
	})

	t.Run("multiple complete cycles", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		// First cycle
		pm.Start()
		time.Sleep(25 * time.Millisecond)
		pm.Stop()
		firstElapsed := pm.ElapsedMilliseconds()

		// Second cycle (without reset, should update endTime)
		pm.Start()
		time.Sleep(25 * time.Millisecond)
		pm.Stop()
		secondElapsed := pm.ElapsedMilliseconds()

		// Third cycle with reset
		pm.Reset()
		pm.Start()
		time.Sleep(25 * time.Millisecond)
		pm.Stop()
		thirdElapsed := pm.ElapsedMilliseconds()

		assert.Greater(t, firstElapsed, 15.0)
		assert.Less(t, firstElapsed, 50.0)
		assert.Greater(t, secondElapsed, 15.0)
		assert.Less(t, secondElapsed, 50.0)
		assert.Greater(t, thirdElapsed, 15.0)
		assert.Less(t, thirdElapsed, 50.0)
	})

	t.Run("very short duration measurement", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Start()
		// No sleep, just immediate stop
		pm.Stop()

		elapsed := pm.ElapsedMilliseconds()

		// Should be very small but non-zero (or zero if too fast)
		assert.GreaterOrEqual(t, elapsed, 0.0)
		assert.Less(t, elapsed, 10.0)
	})

	t.Run("longer duration measurement", func(t *testing.T) {
		pm := NewPerformanceMonitor()

		pm.Start()
		time.Sleep(200 * time.Millisecond)
		pm.Stop()

		elapsed := pm.ElapsedMilliseconds()

		assert.Greater(t, elapsed, 180.0)
		assert.Less(t, elapsed, 250.0)
	})
}
