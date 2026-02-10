package cacher

import (
	"context"
	"time"
)

// FetchFunc is a function that fetches a value from the source when a cache miss occurs.
// It receives a context for cancellation and timeout control, and returns the value
// of type T or an error if the fetch operation fails.
type FetchFunc[T any] func(ctx context.Context) (T, error)

// Cacher is an interface that defines methods for caching values with automatic
// fetching on cache misses. Implementations should provide thread-safe caching
// and handle cache stampede prevention when multiple concurrent requests occur
// for the same missing cache key.
type Cacher[T any] interface {
	// GetOrFetch retrieves a value from the cache, or fetches it using the provided
	// function if it's not cached. The fetched value is then stored in the cache
	// with the specified TTL for future requests.
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
	GetOrFetch(
		ctx context.Context,
		key string,
		ttl time.Duration,
		fetchFn FetchFunc[T],
	) (T, error)

	// Delete removes a key from the cache.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - key: The cache key to delete
	Delete(ctx context.Context, key string) error

	// Clear removes all items from the cache.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//
	// Returns:
	//   - An error if the operation fails
	Clear(ctx context.Context) error

	// ItemCount returns the number of items in the cache.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//
	// Returns:
	//   - The number of items in the cache
	//   - An error if the operation fails
	ItemCount(ctx context.Context) (int, error)

	// DeleteByPrefix deletes all keys with the given prefix.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - prefix: The prefix to match keys against
	//
	// Returns:
	//   - The number of keys deleted
	//   - An error if the operation fails
	DeleteByPrefix(ctx context.Context, prefix string) (int, error)
}
