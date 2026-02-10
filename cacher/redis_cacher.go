package cacher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisCacher is a Redis-based implementation of the Cacher interface.
// It provides thread-safe caching with distributed locking to prevent
// cache stampede (thundering herd) problems when multiple goroutines
// try to fetch the same missing cache entry simultaneously.
type redisCacher[T any] struct {
	client *redis.Client
}

// NewRedisCacher creates a new Redis-based cacher instance.
// It takes a Redis client and returns a Cacher implementation that
// uses Redis for storage and distributed locking.
//
// Example:
//
//	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	cacher := NewRedisCacher[string](client)
func NewRedisCacher[T any](client *redis.Client) Cacher[T] {
	return &redisCacher[T]{
		client: client,
	}
}

// GetOrFetch retrieves a value from the cache, or fetches it using the provided
// function if it's not cached. It implements distributed locking to prevent
// cache stampede when multiple goroutines request the same missing key.
//
// The method works as follows:
//  1. First attempts to retrieve the value from Redis cache
//  2. On cache miss, attempts to acquire a distributed lock
//  3. If lock is acquired, fetches the value, caches it, and releases the lock
//  4. If lock acquisition fails, waits for another goroutine to populate the cache
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
//
// The lock is automatically extended if the fetch operation takes longer than
// the initial lock TTL (30 seconds), and is safely released using a Lua script
// that verifies lock ownership.
func (c *redisCacher[T]) GetOrFetch(ctx context.Context, key string, ttl time.Duration, fetchFn FetchFunc[T]) (T, error) {
	var zero T

	// Try to get from cache first
	val, err := c.client.Get(ctx, key).Result()
	if err == nil {
		var result T
		if err := json.Unmarshal([]byte(val), &result); err != nil {
			return zero, fmt.Errorf("failed to unmarshal cached value: %w", err)
		}

		return result, nil
	}

	if !errors.Is(err, redis.Nil) {
		return zero, fmt.Errorf("redis get error: %w", err)
	}

	// Cache miss - try to acquire lock
	lockKey := fmt.Sprintf("%s:lock", key)
	lockTTL := 30 * time.Second
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano()) // Unique lock value

	acquired, err := c.client.SetNX(ctx, lockKey, lockValue, lockTTL).Result()
	if err != nil {
		return zero, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if acquired {
		// Use background context for cleanup to ensure lock is released
		bgCtx := context.Background()
		defer func() {
			// Only delete if we still own the lock
			script := `
				if redis.call("get", KEYS[1]) == ARGV[1] then
					return redis.call("del", KEYS[1])
				else
					return 0
				end
			`
			c.client.Eval(bgCtx, script, []string{lockKey}, lockValue)
		}()

		// Extend lock if fetch takes longer
		extendCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go c.extendLock(extendCtx, lockKey, lockValue, lockTTL)

		result, err := fetchFn(ctx)
		if err != nil {
			return zero, fmt.Errorf("fetch function failed: %w", err)
		}

		data, err := json.Marshal(result)
		if err != nil {
			return zero, fmt.Errorf("failed to marshal result: %w", err)
		}

		// Set cache value
		if err := c.client.Set(bgCtx, key, data, ttl).Err(); err != nil {
			return zero, fmt.Errorf("failed to cache result: %w", err)
		}

		return result, nil
	}

	// Another goroutine is fetching - wait for result
	return c.waitForCache(ctx, key, lockKey, 30*time.Second)
}

// extendLock periodically extends the lock TTL to prevent expiration
// during long-running fetch operations. It runs in a separate goroutine
// and extends the lock at intervals of ttl/3 until the context is cancelled.
//
// The extension uses a Lua script to atomically verify lock ownership
// before extending, ensuring only the lock owner can extend it.
//
// Parameters:
//   - ctx: Context for cancellation (when cancelled, extension stops)
//   - lockKey: The Redis key for the lock
//   - lockValue: The unique value identifying this lock instance
//   - ttl: The time-to-live duration to extend the lock to
func (c *redisCacher[T]) extendLock(ctx context.Context, lockKey, lockValue string, ttl time.Duration) {
	ticker := time.NewTicker(ttl / 3) // Extend at 1/3 of TTL
	defer ticker.Stop()

	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.client.Eval(ctx, script, []string{lockKey}, lockValue, ttl.Milliseconds())
		}
	}
}

// waitForCache waits for another goroutine to populate the cache after
// failing to acquire the lock. It uses exponential backoff polling to
// efficiently check for the cached value while respecting context cancellation
// and timeout limits.
//
// The method polls the cache with exponential backoff (starting at 10ms,
// doubling up to 500ms max) until:
//   - The value appears in cache (success)
//   - The lock disappears without a cached value (fetch likely failed)
//   - The timeout is reached
//   - The context is cancelled
//
// Parameters:
//   - ctx: Context for cancellation control
//   - key: The cache key to wait for
//   - lockKey: The lock key to monitor
//   - timeout: Maximum duration to wait for the cache value
//
// Returns:
//   - The cached value of type T if found
//   - An error if timeout occurs, context is cancelled, or fetch operation failed
func (c *redisCacher[T]) waitForCache(
	ctx context.Context,
	key string,
	lockKey string,
	timeout time.Duration,
) (T, error) {
	var zero T

	// Use exponential backoff instead of fixed polling
	backoff := 10 * time.Millisecond
	maxBackoff := 500 * time.Millisecond
	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}

		if time.Now().After(deadline) {
			return zero, errors.New("timeout waiting for cache")
		}

		// Check if value is in cache
		val, err := c.client.Get(ctx, key).Result()
		if err == nil {
			var result T
			if err := json.Unmarshal([]byte(val), &result); err != nil {
				return zero, fmt.Errorf("failed to unmarshal cached value: %w", err)
			}

			return result, nil
		}

		if !errors.Is(err, redis.Nil) {
			return zero, fmt.Errorf("redis get error: %w", err)
		}

		// Check if lock still exists
		exists, err := c.client.Exists(ctx, lockKey).Result()
		if err != nil {
			return zero, fmt.Errorf("failed to check lock existence: %w", err)
		}

		if exists == 0 {
			// Lock is gone but no cached value - fetch operation likely failed
			// Try one more time to get from cache in case of timing issue
			val, err := c.client.Get(ctx, key).Result()
			if err == nil {
				var result T
				if err := json.Unmarshal([]byte(val), &result); err != nil {
					return zero, fmt.Errorf("failed to unmarshal cached value: %w", err)
				}
				return result, nil
			}
			return zero, errors.New("fetch operation failed or cache not populated")
		}

		// Exponential backoff
		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// Delete removes a key from the cache.
func (c *redisCacher[T]) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}
	return nil
}

// Clear removes all items from the cache.
func (c *redisCacher[T]) Clear(ctx context.Context) error {
	if err := c.client.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}
	return nil
}

// ItemCount returns the number of items in the cache.
func (c *redisCacher[T]) ItemCount(ctx context.Context) (int, error) {
	count, err := c.client.DBSize(ctx).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get cache size: %w", err)
	}
	return int(count), nil
}

// DeleteByPrefix deletes all keys with the given prefix.
func (c *redisCacher[T]) DeleteByPrefix(ctx context.Context, prefix string) (int, error) {
	deletedCount := 0

	// Use SCAN to iterate through keys with the prefix
	// This is more efficient than KEYS for large datasets
	iter := c.client.Scan(ctx, 0, prefix+"*", 0).Iterator()
	var keysToDelete []string

	for iter.Next(ctx) {
		// Check context cancellation during iteration
		select {
		case <-ctx.Done():
			return deletedCount, ctx.Err()
		default:
		}

		key := iter.Val()
		if strings.HasPrefix(key, prefix) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	if err := iter.Err(); err != nil {
		return deletedCount, fmt.Errorf("failed to scan keys: %w", err)
	}

	// Delete keys in batches for efficiency
	if len(keysToDelete) > 0 {
		deleted, err := c.client.Del(ctx, keysToDelete...).Result()
		if err != nil {
			return deletedCount, fmt.Errorf("failed to delete keys: %w", err)
		}
		deletedCount = int(deleted)
	}

	return deletedCount, nil
}
