package idgenerator

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIdGenerator(t *testing.T) {
	t.Run("returns non-nil generator", func(t *testing.T) {
		gen := NewIdGenerator(0)
		require.NotNil(t, gen)
	})

	t.Run("first Id returns startValue+1 when startValue is 0", func(t *testing.T) {
		gen := NewIdGenerator(0)
		got := gen.Id()
		assert.Equal(t, uint32(1), got)
	})

	t.Run("first Id returns startValue+1 when startValue is non-zero", func(t *testing.T) {
		gen := NewIdGenerator(100)
		got := gen.Id()
		assert.Equal(t, uint32(101), got)
	})

	t.Run("first Id returns 1 when startValue is max uint32 (overflow next is 0)", func(t *testing.T) {
		gen := NewIdGenerator(^uint32(0)) // max uint32
		got := gen.Id()
		assert.Equal(t, uint32(0), got) // 0 after overflow
	})
}

func TestIdGenerator_Id_sequential(t *testing.T) {
	t.Run("ids are monotonic starting from 1", func(t *testing.T) {
		gen := NewIdGenerator(0)
		for want := uint32(1); want <= 10; want++ {
			got := gen.Id()
			assert.Equal(t, want, got)
		}
	})

	t.Run("ids are monotonic with custom start", func(t *testing.T) {
		gen := NewIdGenerator(1000)
		for i := 0; i < 5; i++ {
			got := gen.Id()
			assert.Equal(t, uint32(1001+i), got)
		}
	})

	t.Run("no duplicate ids in sequence", func(t *testing.T) {
		gen := NewIdGenerator(0)
		seen := make(map[uint32]bool)
		for i := 0; i < 100; i++ {
			id := gen.Id()
			assert.False(t, seen[id], "duplicate id %d", id)
			seen[id] = true
		}
	})
}

func TestIdGenerator_Id_concurrent(t *testing.T) {
	t.Run("concurrent Id calls produce unique ids", func(t *testing.T) {
		gen := NewIdGenerator(0)
		const n = 500
		ids := make([]uint32, n)
		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func(idx int) {
				defer wg.Done()
				ids[idx] = gen.Id()
			}(i)
		}
		wg.Wait()

		seen := make(map[uint32]bool)
		for _, id := range ids {
			assert.False(t, seen[id], "duplicate id %d", id)
			seen[id] = true
		}
		assert.Len(t, seen, n)
	})

	t.Run("concurrent Id calls are monotonic when collected", func(t *testing.T) {
		gen := NewIdGenerator(0)
		const n = 200
		ids := make([]uint32, n)
		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func(idx int) {
				defer wg.Done()
				ids[idx] = gen.Id()
			}(i)
		}
		wg.Wait()

		// All IDs should be in range [1, n]
		for _, id := range ids {
			assert.GreaterOrEqual(t, id, uint32(1))
			assert.LessOrEqual(t, id, uint32(n))
		}
	})
}

func TestIdGenerator_multiple_generators_independent(t *testing.T) {
	gen1 := NewIdGenerator(0)
	gen2 := NewIdGenerator(0)

	id1 := gen1.Id()
	id2 := gen2.Id()
	assert.Equal(t, uint32(1), id1)
	assert.Equal(t, uint32(1), id2)

	// Each generator has its own sequence
	assert.Equal(t, uint32(2), gen1.Id())
	assert.Equal(t, uint32(2), gen2.Id())
}

func TestIdGenerator_reserve_zero(t *testing.T) {
	// Documented use case: start from 0 so first ID is 1 and 0 can mean "invalid"
	gen := NewIdGenerator(0)
	assert.Equal(t, uint32(1), gen.Id(), "first id should be 1 when reserving 0")
}
