package safemap

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSafeMap(t *testing.T) {
	m := NewSafeMap[string, int]()
	require.NotNil(t, m)
	assert.Equal(t, 0, m.Len())
	_, ok := m.Load("x")
	assert.False(t, ok)
}

func TestSafeMap_Store_Load(t *testing.T) {
	m := NewSafeMap[string, int]()

	t.Run("store and load returns value", func(t *testing.T) {
		m.Store("a", 1)
		v, ok := m.Load("a")
		assert.True(t, ok)
		assert.Equal(t, 1, v)
	})

	t.Run("overwrite returns new value", func(t *testing.T) {
		m.Store("a", 2)
		v, ok := m.Load("a")
		assert.True(t, ok)
		assert.Equal(t, 2, v)
	})

	t.Run("load missing key returns zero value and false", func(t *testing.T) {
		v, ok := m.Load("nonexistent")
		assert.False(t, ok)
		assert.Equal(t, 0, v)
	})
}

func TestSafeMap_Set_Get(t *testing.T) {
	m := NewSafeMap[string, string]()

	t.Run("set and get", func(t *testing.T) {
		m.Set("k", "v")
		v, ok := m.Get("k")
		assert.True(t, ok)
		assert.Equal(t, "v", v)
	})

	t.Run("get missing key", func(t *testing.T) {
		v, ok := m.Get("missing")
		assert.False(t, ok)
		assert.Empty(t, v)
	})
}

func TestSafeMap_Delete(t *testing.T) {
	m := NewSafeMap[string, int]()
	m.Store("a", 1)
	m.Store("b", 2)

	t.Run("delete removes key", func(t *testing.T) {
		m.Delete("a")
		_, ok := m.Load("a")
		assert.False(t, ok)
		v, ok := m.Load("b")
		assert.True(t, ok)
		assert.Equal(t, 2, v)
	})

	t.Run("delete missing key is no-op", func(t *testing.T) {
		m.Delete("nonexistent")
		assert.Equal(t, 1, m.Len())
	})
}

func TestSafeMap_Has(t *testing.T) {
	m := NewSafeMap[int, struct{}]()
	m.Store(1, struct{}{})

	assert.True(t, m.Has(1))
	assert.False(t, m.Has(2))
	m.Delete(1)
	assert.False(t, m.Has(1))
}

func TestSafeMap_Len(t *testing.T) {
	m := NewSafeMap[string, int]()

	assert.Equal(t, 0, m.Len())
	m.Store("a", 1)
	assert.Equal(t, 1, m.Len())
	m.Store("b", 2)
	assert.Equal(t, 2, m.Len())
	m.Delete("a")
	assert.Equal(t, 1, m.Len())
	m.Delete("b")
	assert.Equal(t, 0, m.Len())
}

func TestSafeMap_Range(t *testing.T) {
	m := NewSafeMap[string, int]()
	m.Store("a", 1)
	m.Store("b", 2)
	m.Store("c", 3)

	t.Run("iterates all entries", func(t *testing.T) {
		seen := make(map[string]int)
		m.Range(func(k string, v int) bool {
			seen[k] = v
			return true
		})
		assert.Len(t, seen, 3)
		assert.Equal(t, 1, seen["a"])
		assert.Equal(t, 2, seen["b"])
		assert.Equal(t, 3, seen["c"])
	})

	t.Run("stops when f returns false", func(t *testing.T) {
		count := 0
		m.Range(func(k string, v int) bool {
			count++
			return count < 2
		})
		assert.Equal(t, 2, count)
	})

	t.Run("empty map calls f zero times", func(t *testing.T) {
		empty := NewSafeMap[string, int]()
		calls := 0
		empty.Range(func(k string, v int) bool {
			calls++
			return true
		})
		assert.Equal(t, 0, calls)
	})
}

func TestSafeMap_ZeroValueType(t *testing.T) {
	t.Run("pointer value zero is nil", func(t *testing.T) {
		m := NewSafeMap[string, *int]()
		v, ok := m.Load("x")
		assert.False(t, ok)
		assert.Nil(t, v)
	})

	t.Run("slice value zero is nil", func(t *testing.T) {
		m := NewSafeMap[string, []byte]()
		v, ok := m.Load("x")
		assert.False(t, ok)
		assert.Nil(t, v)
	})
}

func TestSafeMap_Concurrent(t *testing.T) {
	m := NewSafeMap[int, int]()
	const goroutines = 100
	const opsPerGoroutine = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := range goroutines {
		go func(id int) {
			defer wg.Done()
			for i := range opsPerGoroutine {
				key := id*opsPerGoroutine + i
				m.Store(key, key*2)
				m.Load(key)
				m.Has(key)
			}
		}(g)
	}
	wg.Wait()

	assert.Equal(t, goroutines*opsPerGoroutine, m.Len())

	// Concurrent delete and read
	wg.Add(goroutines)
	for g := range goroutines {
		go func(id int) {
			defer wg.Done()
			for i := range opsPerGoroutine {
				key := id*opsPerGoroutine + i
				m.Delete(key)
				m.Load(key)
				m.Len()
			}
		}(g)
	}
	wg.Wait()

	assert.Equal(t, 0, m.Len())
}

func TestSafeMap_ConcurrentRange(t *testing.T) {
	m := NewSafeMap[int, int]()
	for i := range 100 {
		m.Store(i, i)
	}

	// Range while others are reading (should not race)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		count := 0
		m.Range(func(k, v int) bool {
			count++
			return true
		})
		assert.Equal(t, 100, count)
	}()
	go func() {
		defer wg.Done()
		for i := range 100 {
			m.Load(i)
		}
	}()
	wg.Wait()
}
