package cacher

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRedisCacher(t *testing.T, maxItems int) (*redisCacher[string], context.Context) {
	t.Helper()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		require.NoError(t, client.Close())
	})

	c, ok := NewRedisCacher[string](client, maxItems).(*redisCacher[string])
	require.True(t, ok)

	return c, context.Background()
}

func TestRedisCacher_GetOrFetch_TTLZeroBypassesCache(t *testing.T) {
	c, ctx := newTestRedisCacher(t, 2)

	_, err := c.GetOrFetch(ctx, "key", time.Minute, func(ctx context.Context) (string, error) {
		return "cached", nil
	})
	require.NoError(t, err)

	fetchCount := 0
	val, err := c.GetOrFetch(ctx, "key", 0, func(ctx context.Context) (string, error) {
		fetchCount++
		return "fresh", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "fresh", val)
	assert.Equal(t, 1, fetchCount)

	val, err = c.GetOrFetch(ctx, "key", time.Minute, func(ctx context.Context) (string, error) {
		return "miss", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "cached", val)
}

func TestRedisCacher_GetOrFetch_EvictsLeastRecentlyUsed(t *testing.T) {
	c, ctx := newTestRedisCacher(t, 2)

	_, err := c.GetOrFetch(ctx, "a", time.Minute, func(ctx context.Context) (string, error) { return "a1", nil })
	require.NoError(t, err)
	_, err = c.GetOrFetch(ctx, "b", time.Minute, func(ctx context.Context) (string, error) { return "b1", nil })
	require.NoError(t, err)

	val, err := c.GetOrFetch(ctx, "a", time.Minute, func(ctx context.Context) (string, error) { return "a2", nil })
	require.NoError(t, err)
	assert.Equal(t, "a1", val)

	_, err = c.GetOrFetch(ctx, "c", time.Minute, func(ctx context.Context) (string, error) { return "c1", nil })
	require.NoError(t, err)

	val, err = c.GetOrFetch(ctx, "a", time.Minute, func(ctx context.Context) (string, error) { return "a2", nil })
	require.NoError(t, err)
	assert.Equal(t, "a1", val)

	val, err = c.GetOrFetch(ctx, "b", time.Minute, func(ctx context.Context) (string, error) { return "b2", nil })
	require.NoError(t, err)
	assert.Equal(t, "b2", val)
}

func TestRedisCacher_DeleteAndDeleteByPrefixRemoveLRUMetadata(t *testing.T) {
	c, ctx := newTestRedisCacher(t, 4)

	fetchFn := func(value string) FetchFunc[string] {
		return func(ctx context.Context) (string, error) { return value, nil }
	}

	_, err := c.GetOrFetch(ctx, "user:1", time.Minute, fetchFn("u1"))
	require.NoError(t, err)
	_, err = c.GetOrFetch(ctx, "user:2", time.Minute, fetchFn("u2"))
	require.NoError(t, err)
	_, err = c.GetOrFetch(ctx, "order:1", time.Minute, fetchFn("o1"))
	require.NoError(t, err)

	require.NoError(t, c.Delete(ctx, "user:1"))
	n, err := c.DeleteByPrefix(ctx, "user:")
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	count, err := c.ItemCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	_, err = c.GetOrFetch(ctx, "a", time.Minute, fetchFn("a1"))
	require.NoError(t, err)
	_, err = c.GetOrFetch(ctx, "b", time.Minute, fetchFn("b1"))
	require.NoError(t, err)
	_, err = c.GetOrFetch(ctx, "c", time.Minute, fetchFn("c1"))
	require.NoError(t, err)
	_, err = c.GetOrFetch(ctx, "d", time.Minute, fetchFn("d1"))
	require.NoError(t, err)

	val, err := c.GetOrFetch(ctx, "order:1", time.Minute, fetchFn("o2"))
	require.NoError(t, err)
	assert.Equal(t, "o2", val)
}

func TestRedisCacher_ItemCountIgnoresInternalLRUKeys(t *testing.T) {
	c, ctx := newTestRedisCacher(t, 2)

	_, err := c.GetOrFetch(ctx, "a", time.Minute, func(ctx context.Context) (string, error) { return "a1", nil })
	require.NoError(t, err)
	_, err = c.GetOrFetch(ctx, "b", time.Minute, func(ctx context.Context) (string, error) { return "b1", nil })
	require.NoError(t, err)

	count, err := c.ItemCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
