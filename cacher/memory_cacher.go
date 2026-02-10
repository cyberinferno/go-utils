package cacher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/sync/singleflight"
)

// MemoryCacher is an in-memory implementation of the Cacher interface.
// It uses go-cache for storage and singleflight to prevent cache stampede
// (thundering herd problem) when multiple concurrent requests occur for the
// same cache key.
type MemoryCacher[T any] struct {
	cache *cache.Cache
	group singleflight.Group
}

// NewMemoryCacher creates a new in-memory cache instance with the specified
// default expiration and cleanup interval.
//
// Parameters:
//   - defaultExpiration: Default TTL for cached items (use cache.NoExpiration for no default)
//   - cleanupInterval: Interval at which expired items are removed from the cache
//
// Returns:
//   - A new InMemoryCacher instance
func NewMemoryCacher[T any](defaultExpiration, cleanupInterval time.Duration) Cacher[T] {
	return &MemoryCacher[T]{
		cache: cache.New(defaultExpiration, cleanupInterval),
		group: singleflight.Group{},
	}
}

// GetOrFetch retrieves a value from the cache, or fetches it using the provided
// function if it's not cached. The singleflight group ensures that for concurrent
// requests to the same key, only one fetch operation is executed.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - key: The cache key to retrieve or set
//   - ttl: Time-to-live duration for the cached value
//   - fetchFn: Function to fetch the value if not in cache
//
// Returns:
//   - The cached or fetched value of type T
//   - An error if retrieval or fetching fails
func (c *MemoryCacher[T]) GetOrFetch(
	ctx context.Context,
	key string,
	ttl time.Duration,
	fetchFn FetchFunc[T],
) (T, error) {
	var zero T

	// Try to get from cache first
	if val, found := c.cache.Get(key); found {
		if typedVal, ok := val.(T); ok {
			return typedVal, nil
		}
	}

	// Use singleflight to prevent thundering herd
	// Only one fetch will be executed for concurrent requests with the same key
	val, err, _ := c.group.Do(key, func() (interface{}, error) {
		// Double-check cache after acquiring singleflight lock
		// Another goroutine might have already populated it
		if cachedVal, found := c.cache.Get(key); found {
			if typedVal, ok := cachedVal.(T); ok {
				return typedVal, nil
			}
		}

		// Fetch the value
		fetchedVal, err := fetchFn(ctx)
		if err != nil {
			return zero, err
		}

		// Store in cache with specified TTL
		c.cache.Set(key, fetchedVal, ttl)

		return fetchedVal, nil
	})

	if err != nil {
		return zero, err
	}

	// Type assert the result
	typedVal, ok := val.(T)
	if !ok {
		return zero, fmt.Errorf("unexpected type in cache for key %s", key)
	}

	return typedVal, nil
}

// Delete removes a key from the cache.
func (c *MemoryCacher[T]) Delete(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	c.cache.Delete(key)
	return nil
}

// Clear removes all items from the cache.
func (c *MemoryCacher[T]) Clear(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	c.cache.Flush()
	return nil
}

// ItemCount returns the number of items in the cache.
func (c *MemoryCacher[T]) ItemCount(ctx context.Context) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	return c.cache.ItemCount(), nil
}

// DeleteByPrefix deletes all keys with the given prefix.
func (c *MemoryCacher[T]) DeleteByPrefix(ctx context.Context, prefix string) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	items := c.cache.Items()
	deletedCount := 0

	for key := range items {
		// Check context cancellation during iteration
		select {
		case <-ctx.Done():
			return deletedCount, ctx.Err()
		default:
		}

		if strings.HasPrefix(key, prefix) {
			c.cache.Delete(key)
			deletedCount++
		}
	}

	return deletedCount, nil
}
