package safeset

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSafeSet(t *testing.T) {
	s := NewSafeSet[string]()
	require.NotNil(t, s)
	assert.Equal(t, 0, s.Size())
	assert.False(t, s.Contains("x"))
}

func TestSafeSet_Add_Contains(t *testing.T) {
	s := NewSafeSet[string]()

	t.Run("add and contains returns true", func(t *testing.T) {
		s.Add("a")
		assert.True(t, s.Contains("a"))
		assert.Equal(t, 1, s.Size())
	})

	t.Run("adding duplicate does not increase size", func(t *testing.T) {
		s.Add("a")
		s.Add("a")
		assert.True(t, s.Contains("a"))
		assert.Equal(t, 1, s.Size())
	})

	t.Run("contains missing returns false", func(t *testing.T) {
		assert.False(t, s.Contains("nonexistent"))
	})
}

func TestSafeSet_Remove(t *testing.T) {
	s := NewSafeSet[string]()
	s.Add("a")
	s.Add("b")

	t.Run("remove removes element", func(t *testing.T) {
		s.Remove("a")
		assert.False(t, s.Contains("a"))
		assert.True(t, s.Contains("b"))
		assert.Equal(t, 1, s.Size())
	})

	t.Run("remove missing is no-op", func(t *testing.T) {
		s.Remove("nonexistent")
		assert.Equal(t, 1, s.Size())
	})
}

func TestSafeSet_Size(t *testing.T) {
	s := NewSafeSet[int]()

	assert.Equal(t, 0, s.Size())
	s.Add(1)
	assert.Equal(t, 1, s.Size())
	s.Add(2)
	assert.Equal(t, 2, s.Size())
	s.Add(1) // duplicate
	assert.Equal(t, 2, s.Size())
	s.Remove(1)
	assert.Equal(t, 1, s.Size())
	s.Remove(2)
	assert.Equal(t, 0, s.Size())
}

func TestSafeSet_Intersection(t *testing.T) {
	t.Run("non-empty intersection", func(t *testing.T) {
		a := NewSafeSet[int]()
		a.Add(1)
		a.Add(2)
		a.Add(3)
		b := NewSafeSet[int]()
		b.Add(2)
		b.Add(3)
		b.Add(4)

		got := a.Intersection(b)
		require.NotNil(t, got)
		assert.Equal(t, 2, got.Size())
		assert.True(t, got.Contains(2))
		assert.True(t, got.Contains(3))
		assert.False(t, got.Contains(1))
		assert.False(t, got.Contains(4))
	})

	t.Run("empty intersection", func(t *testing.T) {
		a := NewSafeSet[string]()
		a.Add("x")
		b := NewSafeSet[string]()
		b.Add("y")

		got := a.Intersection(b)
		require.NotNil(t, got)
		assert.Equal(t, 0, got.Size())
		assert.False(t, got.Contains("x"))
		assert.False(t, got.Contains("y"))
	})

	t.Run("one set empty", func(t *testing.T) {
		a := NewSafeSet[int]()
		a.Add(1)
		a.Add(2)
		b := NewSafeSet[int]()

		got := a.Intersection(b)
		require.NotNil(t, got)
		assert.Equal(t, 0, got.Size())
	})

	t.Run("both sets empty", func(t *testing.T) {
		a := NewSafeSet[int]()
		b := NewSafeSet[int]()

		got := a.Intersection(b)
		require.NotNil(t, got)
		assert.Equal(t, 0, got.Size())
	})
}

func TestSafeSet_Union(t *testing.T) {
	t.Run("union of two non-empty sets", func(t *testing.T) {
		a := NewSafeSet[int]()
		a.Add(1)
		a.Add(2)
		b := NewSafeSet[int]()
		b.Add(2)
		b.Add(3)

		got := a.Union(b)
		require.NotNil(t, got)
		assert.Equal(t, 3, got.Size())
		assert.True(t, got.Contains(1))
		assert.True(t, got.Contains(2))
		assert.True(t, got.Contains(3))
	})

	t.Run("one set empty", func(t *testing.T) {
		a := NewSafeSet[string]()
		a.Add("a")
		a.Add("b")
		b := NewSafeSet[string]()

		got := a.Union(b)
		require.NotNil(t, got)
		assert.Equal(t, 2, got.Size())
		assert.True(t, got.Contains("a"))
		assert.True(t, got.Contains("b"))
	})

	t.Run("both sets empty", func(t *testing.T) {
		a := NewSafeSet[int]()
		b := NewSafeSet[int]()

		got := a.Union(b)
		require.NotNil(t, got)
		assert.Equal(t, 0, got.Size())
	})
}

func TestSafeSet_Reset(t *testing.T) {
	s := NewSafeSet[int]()
	s.Add(1)
	s.Add(2)
	assert.Equal(t, 2, s.Size())

	s.Reset()
	assert.Equal(t, 0, s.Size())
	assert.False(t, s.Contains(1))
	assert.False(t, s.Contains(2))

	// Can use after reset
	s.Add(3)
	assert.Equal(t, 1, s.Size())
	assert.True(t, s.Contains(3))
}

func TestSafeSet_Range(t *testing.T) {
	s := NewSafeSet[string]()
	s.Add("a")
	s.Add("b")
	s.Add("c")

	t.Run("iterates all elements", func(t *testing.T) {
		seen := make(map[string]bool)
		s.Range(func(v string) bool {
			seen[v] = true
			return true
		})
		assert.Len(t, seen, 3)
		assert.True(t, seen["a"])
		assert.True(t, seen["b"])
		assert.True(t, seen["c"])
	})

	t.Run("stops when f returns false", func(t *testing.T) {
		count := 0
		s.Range(func(v string) bool {
			count++
			return count < 2
		})
		assert.Equal(t, 2, count)
	})

	t.Run("empty set calls f zero times", func(t *testing.T) {
		empty := NewSafeSet[int]()
		calls := 0
		empty.Range(func(v int) bool {
			calls++
			return true
		})
		assert.Equal(t, 0, calls)
	})
}

func TestSafeSet_Concurrent(t *testing.T) {
	s := NewSafeSet[int]()
	const goroutines = 100
	const opsPerGoroutine = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := range goroutines {
		go func(id int) {
			defer wg.Done()
			for i := range opsPerGoroutine {
				v := id*opsPerGoroutine + i
				s.Add(v)
				s.Contains(v)
				s.Size()
			}
		}(g)
	}
	wg.Wait()

	assert.Equal(t, goroutines*opsPerGoroutine, s.Size())

	// Concurrent remove and read
	wg.Add(goroutines)
	for g := range goroutines {
		go func(id int) {
			defer wg.Done()
			for i := range opsPerGoroutine {
				v := id*opsPerGoroutine + i
				s.Remove(v)
				s.Contains(v)
				s.Size()
			}
		}(g)
	}
	wg.Wait()

	assert.Equal(t, 0, s.Size())
}

func TestSafeSet_ConcurrentRange(t *testing.T) {
	s := NewSafeSet[int]()
	for i := range 100 {
		s.Add(i)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		count := 0
		s.Range(func(v int) bool {
			count++
			return true
		})
		assert.Equal(t, 100, count)
	}()
	go func() {
		defer wg.Done()
		for i := range 100 {
			s.Contains(i)
		}
	}()
	wg.Wait()
}

func TestSafeSet_ConcurrentIntersectionUnion(t *testing.T) {
	a := NewSafeSet[int]()
	b := NewSafeSet[int]()
	for i := range 50 {
		a.Add(i)
	}
	for i := 25; i < 75; i++ {
		b.Add(i)
	}

	var wg sync.WaitGroup
	wg.Add(4)
	go func() { defer wg.Done(); _ = a.Intersection(b) }()
	go func() { defer wg.Done(); _ = a.Union(b) }()
	go func() { defer wg.Done(); _ = b.Intersection(a) }()
	go func() { defer wg.Done(); _ = b.Union(a) }()
	wg.Wait()

	// Verify results are correct
	inter := a.Intersection(b)
	assert.Equal(t, 25, inter.Size())
	union := a.Union(b)
	assert.Equal(t, 75, union.Size())
}
