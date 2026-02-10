package cacher

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryCacher(t *testing.T) {
	c := NewMemoryCacher[string](time.Minute, 10*time.Minute)
	require.NotNil(t, c)

	mc, ok := c.(*MemoryCacher[string])
	require.True(t, ok)
	require.NotNil(t, mc.cache)
}

func TestMemoryCacher_GetOrFetch_CacheMiss(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])
	ctx := context.Background()

	fetchCount := 0
	fetchFn := func(ctx context.Context) (string, error) {
		fetchCount++
		return "value", nil
	}

	val, err := c.GetOrFetch(ctx, "key", time.Minute, fetchFn)
	require.NoError(t, err)
	assert.Equal(t, "value", val)
	assert.Equal(t, 1, fetchCount)
}

func TestMemoryCacher_GetOrFetch_CacheHit(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])
	ctx := context.Background()

	fetchCount := 0
	fetchFn := func(ctx context.Context) (string, error) {
		fetchCount++
		return "value", nil
	}

	// Populate cache
	_, err := c.GetOrFetch(ctx, "key", time.Minute, fetchFn)
	require.NoError(t, err)
	assert.Equal(t, 1, fetchCount)

	// Second call should hit cache - fetchFn not called again
	fetchFn2 := func(ctx context.Context) (string, error) {
		fetchCount++
		return "should not be used", nil
	}
	val, err := c.GetOrFetch(ctx, "key", time.Minute, fetchFn2)
	require.NoError(t, err)
	assert.Equal(t, "value", val)
	assert.Equal(t, 1, fetchCount)
}

func TestMemoryCacher_GetOrFetch_FetchError(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])
	ctx := context.Background()

	fetchFn := func(ctx context.Context) (string, error) {
		return "", assert.AnError
	}

	val, err := c.GetOrFetch(ctx, "key", time.Minute, fetchFn)
	assert.ErrorIs(t, err, assert.AnError)
	assert.Empty(t, val)

	// Cache should not contain the key - next GetOrFetch should call fetch again
	fetchCount := 0
	fetchFn2 := func(ctx context.Context) (string, error) {
		fetchCount++
		return "new", nil
	}
	val, err = c.GetOrFetch(ctx, "key", time.Minute, fetchFn2)
	require.NoError(t, err)
	assert.Equal(t, "new", val)
	assert.Equal(t, 1, fetchCount)
}

func TestMemoryCacher_GetOrFetch_ContextCancelled(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Already cancelled

	// When fetchFn respects context and returns error on cancel, we get that error.
	fetchFn := func(ctx context.Context) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			return "value", nil
		}
	}

	val, err := c.GetOrFetch(ctx, "key", time.Minute, fetchFn)
	assert.Error(t, err)
	assert.Empty(t, val)
}

func TestMemoryCacher_GetOrFetch_ConcurrentSameKey_Singleflight(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])
	ctx := context.Background()

	var fetchCount int32
	fetchFn := func(ctx context.Context) (string, error) {
		atomic.AddInt32(&fetchCount, 1)
		time.Sleep(20 * time.Millisecond)
		return "concurrent-value", nil
	}

	const concurrency = 10
	var wg sync.WaitGroup
	results := make([]string, concurrency)
	errs := make([]error, concurrency)

	for i := range concurrency {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			results[i], errs[i] = c.GetOrFetch(ctx, "same-key", time.Minute, fetchFn)
		}()
	}
	wg.Wait()

	// All should get the same value and no error
	for i := range concurrency {
		require.NoError(t, errs[i])
		assert.Equal(t, "concurrent-value", results[i])
	}
	// Fetch should have been called only once due to singleflight
	assert.Equal(t, int32(1), fetchCount)
}

func TestMemoryCacher_GetOrFetch_ConcurrentDifferentKeys(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])
	ctx := context.Background()

	var fetchCount int32
	fetchFn := func(ctx context.Context) (string, error) {
		atomic.AddInt32(&fetchCount, 1)
		return "value", nil
	}

	const n = 5
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		key := string(rune('a' + i))
		go func(k string) {
			defer wg.Done()
			_, _ = c.GetOrFetch(ctx, k, time.Minute, fetchFn)
		}(key)
	}
	wg.Wait()

	assert.Equal(t, int32(n), fetchCount)
}

func TestMemoryCacher_GetOrFetch_WithStructType(t *testing.T) {
	type payload struct {
		ID   int
		Name string
	}

	c := NewMemoryCacher[payload](cache.NoExpiration, time.Minute).(*MemoryCacher[payload])
	ctx := context.Background()

	want := payload{ID: 1, Name: "test"}
	fetchFn := func(ctx context.Context) (payload, error) {
		return want, nil
	}

	val, err := c.GetOrFetch(ctx, "struct-key", time.Minute, fetchFn)
	require.NoError(t, err)
	assert.Equal(t, want, val)
}

func TestMemoryCacher_Delete(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])
	ctx := context.Background()

	// Set a value
	_, err := c.GetOrFetch(ctx, "key", time.Minute, func(ctx context.Context) (string, error) { return "v", nil })
	require.NoError(t, err)

	err = c.Delete(ctx, "key")
	require.NoError(t, err)

	// Should trigger fetch again
	fetchCount := 0
	val, err := c.GetOrFetch(ctx, "key", time.Minute, func(ctx context.Context) (string, error) {
		fetchCount++
		return "new-v", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "new-v", val)
	assert.Equal(t, 1, fetchCount)
}

func TestMemoryCacher_Delete_ContextCancelled(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := c.Delete(ctx, "key")
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMemoryCacher_Delete_NonExistentKey(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])
	ctx := context.Background()

	err := c.Delete(ctx, "nonexistent")
	require.NoError(t, err)
}

func TestMemoryCacher_Clear(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])
	ctx := context.Background()

	fetchFn := func(ctx context.Context) (string, error) { return "v", nil }
	_, _ = c.GetOrFetch(ctx, "k1", time.Minute, fetchFn)
	_, _ = c.GetOrFetch(ctx, "k2", time.Minute, fetchFn)

	count, err := c.ItemCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	err = c.Clear(ctx)
	require.NoError(t, err)

	count, err = c.ItemCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestMemoryCacher_Clear_ContextCancelled(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := c.Clear(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMemoryCacher_ItemCount(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])
	ctx := context.Background()

	count, err := c.ItemCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	fetchFn := func(ctx context.Context) (string, error) { return "v", nil }
	_, _ = c.GetOrFetch(ctx, "k1", time.Minute, fetchFn)
	count, err = c.ItemCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	_, _ = c.GetOrFetch(ctx, "k2", time.Minute, fetchFn)
	count, err = c.ItemCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestMemoryCacher_ItemCount_ContextCancelled(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	count, err := c.ItemCount(ctx)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, count)
}

func TestMemoryCacher_DeleteByPrefix(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])
	ctx := context.Background()

	fetchFn := func(ctx context.Context) (string, error) { return "v", nil }
	_, _ = c.GetOrFetch(ctx, "user:1", time.Minute, fetchFn)
	_, _ = c.GetOrFetch(ctx, "user:2", time.Minute, fetchFn)
	_, _ = c.GetOrFetch(ctx, "order:1", time.Minute, fetchFn)

	n, err := c.DeleteByPrefix(ctx, "user:")
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	count, err := c.ItemCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// "order:1" should still be there
	val, err := c.GetOrFetch(ctx, "order:1", time.Minute, func(ctx context.Context) (string, error) { return "miss", nil })
	require.NoError(t, err)
	assert.Equal(t, "v", val)
}

func TestMemoryCacher_DeleteByPrefix_NoMatch(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])
	ctx := context.Background()

	_, _ = c.GetOrFetch(ctx, "user:1", time.Minute, func(ctx context.Context) (string, error) { return "v", nil })

	n, err := c.DeleteByPrefix(ctx, "other:")
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	count, _ := c.ItemCount(ctx)
	assert.Equal(t, 1, count)
}

func TestMemoryCacher_DeleteByPrefix_EmptyPrefix(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])
	ctx := context.Background()

	_, _ = c.GetOrFetch(ctx, "a", time.Minute, func(ctx context.Context) (string, error) { return "v", nil })
	_, _ = c.GetOrFetch(ctx, "b", time.Minute, func(ctx context.Context) (string, error) { return "v", nil })

	n, err := c.DeleteByPrefix(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	count, _ := c.ItemCount(ctx)
	assert.Equal(t, 0, count)
}

func TestMemoryCacher_DeleteByPrefix_ContextCancelled(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	n, err := c.DeleteByPrefix(ctx, "any:")
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, n)
}

func TestMemoryCacher_DeleteByPrefix_ContextCancelledDuringIteration(t *testing.T) {
	c := NewMemoryCacher[string](cache.NoExpiration, time.Minute).(*MemoryCacher[string])
	ctx := context.Background()

	// Add many keys so iteration has time to be cancelled
	fetchFn := func(ctx context.Context) (string, error) { return "v", nil }
	for i := range 20 {
		key := string(rune('a'+i)) + ":x"
		_, _ = c.GetOrFetch(ctx, key, time.Minute, fetchFn)
	}

	ctx2, cancel := context.WithCancel(ctx)
	// Cancel after a short delay so we might be in the middle of iteration
	go func() {
		time.Sleep(1 * time.Millisecond)
		cancel()
	}()

	n, err := c.DeleteByPrefix(ctx2, "")
	// We might get context.Canceled and some count, or complete with 20
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled)
	}
	_ = n
}

func TestMemoryCacher_Interface(t *testing.T) {
	// Ensure MemoryCacher implements Cacher
	var _ Cacher[string] = (*MemoryCacher[string])(nil)
}
