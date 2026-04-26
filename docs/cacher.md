# Cacher Documentation

The `cacher` package provides a generic, type-safe caching interface with automatic cache population, configurable LRU item limits, and distributed locking to prevent cache stampede (thundering herd) problems. The package includes memory and Redis-based implementations.

## Features

- **Type-Safe Generic Interface**: Works with any Go type using generics
- **Automatic Cache Population**: Fetches and caches values automatically on cache misses
- **Configurable LRU Limits**: Evicts least recently used entries when `maxItems` is exceeded
- **TTL Zero Bypass**: A `ttl` of `0` always calls the fetch function and does not read or write cache
- **Distributed Locking**: Prevents cache stampede when multiple goroutines request the same missing key
- **Lock Extension**: Automatically extends locks during long-running fetch operations
- **Exponential Backoff**: Efficient polling with exponential backoff for waiting goroutines
- **Context Support**: All operations support context for cancellation and timeouts
- **Thread-Safe**: Safe for concurrent use across multiple goroutines

## Installation

```go
import "github.com/cyberinferno/go-utils/cacher"
```

## Creating a Cacher

### Redis-Based Cacher

Create a Redis-based cacher instance using `NewRedisCacher`. You'll need a Redis client from `github.com/redis/go-redis/v9`.

```go
import (
    "github.com/cyberinferno/go-utils/cacher"
    "github.com/redis/go-redis/v9"
)

// Create Redis client
redisClient := redis.NewClient(&redis.Options{
    Addr:     "localhost:6379",
    Password: "", // no password set
    DB:       0,  // use default DB
})

// Create cacher for string values
stringCacher := cacher.NewRedisCacher[string](redisClient, 1000)

// Create cacher for custom types
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
userCacher := cacher.NewRedisCacher[User](redisClient, 1000)
```

### Parameters

- **client**: A `*redis.Client` instance from `github.com/redis/go-redis/v9` configured with your Redis connection settings
- **maxItems**: Maximum number of cached items before LRU eviction. Use `0` or a negative value for unlimited size.

### Memory-Based Cacher

Create an in-memory cacher instance using `NewMemoryCacher`. This implementation uses `go-cache` for storage and is suitable for single-process applications or testing.

```go
import (
    "github.com/cyberinferno/go-utils/cacher"
    "time"
)

// Create memory cacher with default expiration, cleanup interval, and max items
// - defaultExpiration: Default TTL for cached items (use cache.NoExpiration for no default)
// - cleanupInterval: Interval at which expired items are removed from the cache
// - maxItems: Maximum cached items before LRU eviction (<= 0 means unlimited)
memoryCacher := cacher.NewMemoryCacher[string](
    5*time.Minute,  // default expiration
    10*time.Minute, // cleanup interval
	1000,           // max items
)

// Create cacher for custom types
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
userCacher := cacher.NewMemoryCacher[User](
    time.Hour,      // default expiration
    30*time.Minute, // cleanup interval
	1000,           // max items
)
```

### Parameters

- **defaultExpiration**: Default TTL for cached items. Use `cache.NoExpiration` for items that don't expire by default.
- **cleanupInterval**: Interval at which expired items are automatically removed from the cache. Set to `0` to disable automatic cleanup.
- **maxItems**: Maximum number of cached items before LRU eviction. Use `0` or a negative value for unlimited size.

**Note**: Memory cacher is suitable for single-process applications. For distributed systems, use the Redis-based cacher.

## Basic Usage

### Simple Get or Fetch

The `GetOrFetch` method retrieves a value from cache, or fetches it using a provided function if it's not cached.

```go
import (
    "context"
    "fmt"
    "time"
)

ctx := context.Background()

// Define a fetch function
fetchUser := func(ctx context.Context) (User, error) {
    // Simulate fetching from database or API
    return User{
        ID:    1,
        Name:  "John Doe",
        Email: "john@example.com",
    }, nil
}

// Get or fetch user (cached for 1 hour)
user, err := userCacher.GetOrFetch(
    ctx,
    "user:1",
    time.Hour,
    fetchUser,
)
if err != nil {
    log.Fatalf("Failed to get user: %v", err)
}

fmt.Printf("User: %+v\n", user)
```

### Cache Key Naming

Use consistent, descriptive cache keys:

```go
// Good: Descriptive and namespaced
key := fmt.Sprintf("user:%d", userID)
key := fmt.Sprintf("product:%s:details", productID)
key := fmt.Sprintf("session:%s", sessionID)

// Avoid: Generic or ambiguous keys
key := "data"
key := "temp"
```

### TTL (Time-To-Live)

Set appropriate TTL values based on your data freshness requirements:

```go
// Short-lived cache (5 minutes)
data, err := cacher.GetOrFetch(ctx, "key", 5*time.Minute, fetchFn)

// Medium-lived cache (1 hour)
data, err := cacher.GetOrFetch(ctx, "key", time.Hour, fetchFn)

// Long-lived cache (24 hours)
data, err := cacher.GetOrFetch(ctx, "key", 24*time.Hour, fetchFn)

// Bypass cache entirely and always call fetchFn
data, err := cacher.GetOrFetch(ctx, "key", 0, fetchFn)
```

When `ttl` is `0`, `GetOrFetch` does not read an existing cached value, does not write the fetched value back to the cache, and does not update LRU metadata. This is useful for call sites that need the same fetch API but require fresh data.

### LRU Max Item Limits

Both memory and Redis cachers accept `maxItems` in their constructors. When `maxItems` is greater than zero, cache hits and successful cache writes mark a key as recently used. Adding a new item beyond the limit evicts the least recently used cached key.

```go
// Keep at most 1,000 items in Redis.
redisCacher := cacher.NewRedisCacher[User](redisClient, 1000)

// Keep at most 500 items in process memory.
memoryCacher := cacher.NewMemoryCacher[User](
	time.Hour,
	10*time.Minute,
	500,
)

// Disable max-item eviction.
unlimitedCacher := cacher.NewRedisCacher[User](redisClient, 0)
```

Redis LRU tracking is managed by the cacher using internal Redis keys. `Delete`, `DeleteByPrefix`, and `Clear` clean up the associated LRU metadata for managed keys.

## Cache Management

The cacher provides several methods for managing cached data:

### Delete

Removes a specific key from the cache:

```go
// Delete a specific cache key
if err := cacher.Delete(ctx, "user:123"); err != nil {
	log.Printf("Failed to delete cache key: %v", err)
}
```

### Clear

Removes all items from the cache:

```go
// Clear all cached items
if err := cacher.Clear(ctx); err != nil {
	log.Printf("Failed to clear cache: %v", err)
}
```

**Note**: For Redis cacher, this clears all keys in the current database. Use with caution in production environments.

### ItemCount

Returns the number of items currently in the cache:

```go
// Get the number of cached items
count, err := cacher.ItemCount(ctx)
if err != nil {
	log.Printf("Failed to get cache size: %v", err)
}
fmt.Printf("Cache contains %d items\n", count)
```

### DeleteByPrefix

Deletes all keys that start with the given prefix. This is useful for invalidating related cache entries:

```go
// Delete all user-related cache entries
deletedCount, err := cacher.DeleteByPrefix(ctx, "user:")
if err != nil {
	log.Printf("Failed to delete user cache entries: %v", err)
}
fmt.Printf("Deleted %d keys with prefix 'user:'\n", deletedCount)

// Delete all product cache entries
deletedCount, err = cacher.DeleteByPrefix(ctx, "product:")
if err != nil {
	log.Printf("Failed to delete product cache entries: %v", err)
}
fmt.Printf("Deleted %d keys with prefix 'product:'\n", deletedCount)
```

**Example: Cache Invalidation on Data Update**

```go
func (s *UserService) UpdateUser(userID int, updates User) error {
    // Update user in database
    err := s.db.UpdateUser(userID, updates)
    if err != nil {
        return err
    }
    
    // Invalidate specific user cache
    if err := s.cacher.Delete(context.Background(), fmt.Sprintf("user:%d", userID)); err != nil {
        return err
    }

    // Optionally invalidate all related caches
    _, err = s.cacher.DeleteByPrefix(context.Background(), fmt.Sprintf("user:%d:", userID))
    return err
}
```

**Example: Bulk Cache Invalidation**

```go
func (s *ProductService) RefreshAllProducts() error {
    // Clear all product-related caches
    deletedCount, err := s.cacher.DeleteByPrefix(context.Background(), "product:")
    if err != nil {
        return err
    }
    log.Printf("Invalidated %d product cache entries", deletedCount)
    
    // Fetch and cache fresh data
    return s.refreshProducts()
}
```

## Advanced Usage

### Context with Timeout

Use context with timeout to prevent operations from hanging:

```go
import (
    "context"
    "time"
)

// Create context with 10 second timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

user, err := userCacher.GetOrFetch(ctx, "user:1", time.Hour, fetchUser)
if err != nil {
    if err == context.DeadlineExceeded {
        log.Println("Operation timed out")
    } else {
        log.Fatalf("Error: %v", err)
    }
}
```

### Context Cancellation

Cancel operations when needed:

```go
ctx, cancel := context.WithCancel(context.Background())

// Cancel after 5 seconds
go func() {
    time.Sleep(5 * time.Second)
    cancel()
}()

user, err := userCacher.GetOrFetch(ctx, "user:1", time.Hour, fetchUser)
if err != nil {
    if err == context.Canceled {
        log.Println("Operation was cancelled")
    }
}
```

### Error Handling in Fetch Functions

Handle errors properly in your fetch functions:

```go
fetchUser := func(ctx context.Context) (User, error) {
    // Check context cancellation
    select {
    case <-ctx.Done():
        return User{}, ctx.Err()
    default:
    }
    
    // Simulate database query
    user, err := db.GetUser(ctx, userID)
    if err != nil {
        return User{}, fmt.Errorf("failed to fetch user from database: %w", err)
    }
    
    return user, nil
}
```

## How It Works

### Cache Hit Flow

1. Request comes in for a cached key
2. Value is retrieved from Redis cache
3. Value is unmarshaled and returned immediately

### Cache Miss Flow (Single Request)

1. Request comes in for a missing key
2. Attempts to acquire a distributed lock
3. If lock acquired:
   - Fetches value using provided function
   - Stores value in cache with TTL
   - Releases lock
   - Returns value
4. If lock acquisition fails (another goroutine is fetching):
   - Waits for cache to be populated
   - Polls cache with exponential backoff
   - Returns value once available

### Cache Stampede Prevention

When multiple goroutines request the same missing key simultaneously:

1. Only one goroutine acquires the lock and performs the fetch
2. Other goroutines wait and poll the cache
3. Once the first goroutine caches the value, all waiting goroutines retrieve it
4. Lock is automatically extended if fetch takes longer than initial TTL (30 seconds)

## Complete Examples

### Example 1: User Service with Caching

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/cyberinferno/go-utils/cacher"
    "github.com/redis/go-redis/v9"
)

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

type UserService struct {
    cacher cacher.Cacher[User]
    ctx    context.Context
}

func NewUserService(redisClient *redis.Client) *UserService {
    return &UserService{
        cacher: cacher.NewRedisCacher[User](redisClient, 1000),
        ctx:    context.Background(),
    }
}

func (s *UserService) GetUser(userID int) (User, error) {
    key := fmt.Sprintf("user:%d", userID)
    
    fetchUser := func(ctx context.Context) (User, error) {
        // Simulate database query
        // In real implementation, query your database here
        return User{
            ID:    userID,
            Name:  "John Doe",
            Email: "john@example.com",
        }, nil
    }
    
    return s.cacher.GetOrFetch(s.ctx, key, time.Hour, fetchUser)
}

func main() {
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    service := NewUserService(redisClient)
    
    // First call - cache miss, fetches from database
    user1, err := service.GetUser(1)
    if err != nil {
        log.Fatalf("Failed to get user: %v", err)
    }
    fmt.Printf("User 1: %+v\n", user1)
    
    // Second call - cache hit, returns immediately
    user2, err := service.GetUser(1)
    if err != nil {
        log.Fatalf("Failed to get user: %v", err)
    }
    fmt.Printf("User 2: %+v\n", user2)
}
```

### Example 2: API Response Caching

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "time"

    "github.com/cyberinferno/go-utils/cacher"
    "github.com/redis/go-redis/v9"
)

type Product struct {
    ID          int     `json:"id"`
    Name        string  `json:"name"`
    Price       float64 `json:"price"`
    Description string  `json:"description"`
}

type ProductService struct {
    cacher cacher.Cacher[Product]
    apiURL string
    ctx    context.Context
}

func NewProductService(redisClient *redis.Client, apiURL string) *ProductService {
    return &ProductService{
        cacher: cacher.NewRedisCacher[Product](redisClient, 1000),
        apiURL: apiURL,
        ctx:    context.Background(),
    }
}

func (s *ProductService) GetProduct(productID int) (Product, error) {
    key := fmt.Sprintf("product:%d", productID)
    
    fetchProduct := func(ctx context.Context) (Product, error) {
        url := fmt.Sprintf("%s/products/%d", s.apiURL, productID)
        
        req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
        if err != nil {
            return Product{}, err
        }
        
        client := &http.Client{Timeout: 10 * time.Second}
        resp, err := client.Do(req)
        if err != nil {
            return Product{}, err
        }
        defer resp.Body.Close()
        
        if resp.StatusCode != http.StatusOK {
            return Product{}, fmt.Errorf("API returned status %d", resp.StatusCode)
        }
        
        body, err := io.ReadAll(resp.Body)
        if err != nil {
            return Product{}, err
        }
        
        var product Product
        if err := json.Unmarshal(body, &product); err != nil {
            return Product{}, err
        }
        
        return product, nil
    }
    
    // Cache API responses for 15 minutes
    return s.cacher.GetOrFetch(s.ctx, key, 15*time.Minute, fetchProduct)
}

func main() {
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    service := NewProductService(redisClient, "https://api.example.com")
    
    product, err := service.GetProduct(123)
    if err != nil {
        log.Fatalf("Failed to get product: %v", err)
    }
    
    fmt.Printf("Product: %+v\n", product)
}
```

### Example 3: Concurrent Requests (Cache Stampede Prevention)

```go
package main

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"

    "github.com/cyberinferno/go-utils/cacher"
    "github.com/redis/go-redis/v9"
)

type Data struct {
    Value string `json:"value"`
}

func main() {
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    cacher := cacher.NewRedisCacher[Data](redisClient, 1000)
    ctx := context.Background()
    
    var wg sync.WaitGroup
    numGoroutines := 10
    
    fetchData := func(ctx context.Context) (Data, error) {
        // Simulate slow database query
        fmt.Println("Fetching data from database...")
        time.Sleep(2 * time.Second)
        return Data{Value: "cached-data"}, nil
    }
    
    // Launch multiple concurrent requests for the same key
    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            start := time.Now()
            data, err := cacher.GetOrFetch(ctx, "shared-key", time.Hour, fetchData)
            duration := time.Since(start)
            
            if err != nil {
                log.Printf("Goroutine %d failed: %v", id, err)
                return
            }
            
            fmt.Printf("Goroutine %d: Got data in %v: %+v\n", id, duration, data)
        }(i)
    }
    
    wg.Wait()
    fmt.Println("All goroutines completed")
    
    // Output will show:
    // - Only one "Fetching data from database..." message (cache stampede prevented)
    // - All goroutines get the same cached value
    // - Some goroutines wait and retrieve from cache (faster)
}
```

### Example 4: Configuration Caching

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/cyberinferno/go-utils/cacher"
    "github.com/redis/go-redis/v9"
)

type Config struct {
    APIKey     string            `json:"api_key"`
    Endpoints  map[string]string `json:"endpoints"`
    RateLimit  int               `json:"rate_limit"`
    Timeout    int               `json:"timeout"`
}

type ConfigService struct {
    cacher cacher.Cacher[Config]
    ctx    context.Context
}

func NewConfigService(redisClient *redis.Client) *ConfigService {
    return &ConfigService{
        cacher: cacher.NewRedisCacher[Config](redisClient, 1000),
        ctx:    context.Background(),
    }
}

func (s *ConfigService) GetConfig(serviceName string) (Config, error) {
    key := fmt.Sprintf("config:%s", serviceName)
    
    fetchConfig := func(ctx context.Context) (Config, error) {
        // Fetch from configuration service or database
        // This is called only on cache miss
        return Config{
            APIKey: "secret-key",
            Endpoints: map[string]string{
                "api":    "https://api.example.com",
                "cdn":    "https://cdn.example.com",
                "storage": "https://storage.example.com",
            },
            RateLimit: 100,
            Timeout:   30,
        }, nil
    }
    
    // Cache configuration for 1 hour
    return s.cacher.GetOrFetch(s.ctx, key, time.Hour, fetchConfig)
}

func main() {
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    service := NewConfigService(redisClient)
    
    config, err := service.GetConfig("payment-service")
    if err != nil {
        log.Fatalf("Failed to get config: %v", err)
    }
    
    fmt.Printf("Config: %+v\n", config)
}
```

### Example 5: Error Handling

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "time"

    "github.com/cyberinferno/go-utils/cacher"
    "github.com/redis/go-redis/v9"
)

type Item struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

func main() {
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    cacher := cacher.NewRedisCacher[Item](redisClient, 1000)
    ctx := context.Background()
    
    fetchItem := func(ctx context.Context) (Item, error) {
        // Simulate an error condition
        return Item{}, errors.New("item not found in database")
    }
    
    item, err := cacher.GetOrFetch(ctx, "item:999", time.Hour, fetchItem)
    if err != nil {
        // Error from fetch function is returned
        log.Printf("Failed to fetch item: %v", err)
        return
    }
    
    fmt.Printf("Item: %+v\n", item)
}
```

## Best Practices

### 1. Choose Appropriate TTL Values

Select TTL values based on how frequently your data changes:

```go
// Frequently changing data - short TTL
cacher.GetOrFetch(ctx, "rate:USD", 1*time.Minute, fetchRate)

// Moderately changing data - medium TTL
cacher.GetOrFetch(ctx, "user:123", time.Hour, fetchUser)

// Rarely changing data - long TTL
cacher.GetOrFetch(ctx, "config:app", 24*time.Hour, fetchConfig)
```

### 2. Use Descriptive Cache Keys

Use consistent, namespaced cache keys:

```go
// Good: Clear namespace and identifier
key := fmt.Sprintf("user:%d:profile", userID)
key := fmt.Sprintf("product:%s:details", productID)
key := fmt.Sprintf("session:%s:data", sessionID)

// Avoid: Generic or ambiguous keys
key := "data"
key := "temp"
key := "123"
```

### 3. Handle Context Properly

Always use context for cancellation and timeouts:

```go
// Create context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// Pass context to GetOrFetch
data, err := cacher.GetOrFetch(ctx, "key", ttl, fetchFn)
if err != nil {
    if err == context.DeadlineExceeded {
        // Handle timeout
    } else if err == context.Canceled {
        // Handle cancellation
    } else {
        // Handle other errors
    }
}
```

### 4. Make Fetch Functions Idempotent

Ensure your fetch functions can be safely retried:

```go
fetchUser := func(ctx context.Context) (User, error) {
    // This should be safe to call multiple times
    // and return the same result for the same input
    return db.GetUser(ctx, userID)
}
```

### 5. Handle Errors in Fetch Functions

Properly handle and return errors from fetch functions:

```go
fetchData := func(ctx context.Context) (Data, error) {
    // Check context cancellation
    select {
    case <-ctx.Done():
        return Data{}, ctx.Err()
    default:
    }
    
    // Perform fetch operation
    data, err := expensiveOperation(ctx)
    if err != nil {
        // Return error with context
        return Data{}, fmt.Errorf("failed to fetch: %w", err)
    }
    
    return data, nil
}
```

### 6. Use Type-Safe Cachers

Create separate cacher instances for different types:

```go
// String cacher
stringCacher := cacher.NewRedisCacher[string](redisClient, 1000)

// User cacher
userCacher := cacher.NewRedisCacher[User](redisClient, 1000)

// Config cacher
configCacher := cacher.NewRedisCacher[Config](redisClient, 1000)
```

### 7. Monitor Cache Performance

Track cache hit/miss rates and adjust TTL values accordingly:

```go
// Add logging to understand cache behavior
fetchUser := func(ctx context.Context) (User, error) {
    log.Println("Cache miss - fetching from database")
    return db.GetUser(ctx, userID)
}

// Cache hit - no log message (fast path)
user, err := userCacher.GetOrFetch(ctx, key, ttl, fetchUser)
```

### 8. Cache Invalidation Strategies

For data that needs to be invalidated, you have several options:

**Option 1: Delete specific keys when data changes**

```go
// Update user in database
err := db.UpdateUser(userID, updates)
if err != nil {
    return err
}

// Invalidate cache for this user
if err := cacher.Delete(ctx, fmt.Sprintf("user:%d", userID)); err != nil {
    return err
}
```

**Option 2: Delete by prefix for related data**

```go
// Update user profile
err := db.UpdateUserProfile(userID, profile)
if err != nil {
    return err
}

// Invalidate all user-related caches (user:123:profile, user:123:settings, etc.)
if _, err := cacher.DeleteByPrefix(ctx, fmt.Sprintf("user:%d:", userID)); err != nil {
    return err
}
```

**Option 3: Versioned cache keys**

```go
// Versioned cache keys allow easy invalidation
key := fmt.Sprintf("user:%d:v%d", userID, version)

// To invalidate, increment version
version++
newKey := fmt.Sprintf("user:%d:v%d", userID, version)
```

**Option 4: Clear all cache (use with caution)**

```go
// Clear entire cache (useful for testing or major updates)
if err := cacher.Clear(ctx); err != nil {
    return err
}
```

### 9. Handle Redis Connection Errors

Ensure your Redis client is properly configured and handle connection errors:

```go
redisClient := redis.NewClient(&redis.Options{
    Addr:         "localhost:6379",
    PoolSize:     20,
    MinIdleConns: 5,
    MaxRetries:   3,
})

// Test connection
if err := redisClient.Ping(context.Background()).Err(); err != nil {
    log.Fatalf("Failed to connect to Redis: %v", err)
}
```

### 10. Use Appropriate Redis Configuration

Configure Redis client based on your workload:

```go
redisClient := redis.NewClient(&redis.Options{
    Addr:         "localhost:6379",
    Password:     "your-password",
    DB:           0,
    PoolSize:     20,              // Adjust based on concurrency
    MinIdleConns: 5,               // Keep connections warm
    MaxRetries:   3,               // Retry failed commands
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
})
```

## Understanding Cache Stampede Prevention

### The Problem

When multiple goroutines simultaneously request the same missing cache key:
- All goroutines detect a cache miss
- All goroutines attempt to fetch the data
- This causes unnecessary load on the data source (database, API, etc.)
- This is called a "cache stampede" or "thundering herd" problem

### The Solution

The cacher uses distributed locking to prevent cache stampede:

1. **First Request**: Acquires lock, fetches data, caches it, releases lock
2. **Concurrent Requests**: Detect lock exists, wait and poll cache
3. **Lock Extension**: If fetch takes longer than 30 seconds, lock is automatically extended
4. **Safe Release**: Lock is released using Lua script that verifies ownership

### Lock Details

- **Lock TTL**: 30 seconds (initial)
- **Lock Extension**: Automatically extended at 1/3 of TTL intervals
- **Lock Key Format**: `{cache-key}:lock`
- **Lock Value**: Unique timestamp-based value for ownership verification
- **Release**: Uses Lua script to atomically verify ownership before deletion

### Waiting Strategy

Goroutines that fail to acquire the lock use exponential backoff:
- **Initial Backoff**: 10ms
- **Maximum Backoff**: 500ms
- **Backoff Multiplier**: 2x per iteration
- **Timeout**: 30 seconds (configurable in implementation)

## Type Reference

### Cacher Interface

```go
type Cacher[T any] interface {
    GetOrFetch(
        ctx context.Context,
        key string,
        ttl time.Duration,
        fetchFn FetchFunc[T],
    ) (T, error)

	// Delete removes a key from the cache.
	Delete(ctx context.Context, key string) error

	// Clear removes all items from the cache.
	Clear(ctx context.Context) error

	// ItemCount returns the number of items in the cache.
	ItemCount(ctx context.Context) (int, error)

	// DeleteByPrefix deletes all keys with the given prefix.
	// Returns the number of keys deleted.
	DeleteByPrefix(ctx context.Context, prefix string) (int, error)
}
```

The `Cacher` interface defines a generic caching interface that works with any type `T`. It provides methods for retrieving and caching values, as well as managing the cache contents.

### FetchFunc Type

```go
type FetchFunc[T any] func(ctx context.Context) (T, error)
```

`FetchFunc` is a function type that fetches a value of type `T` when a cache miss occurs. It receives a context for cancellation and timeout control.

### NewRedisCacher Function

```go
func NewRedisCacher[T any](client *redis.Client, maxItems int) Cacher[T]
```

Creates a new Redis-based cacher instance. The type parameter `T` determines what type of values will be cached. When `maxItems` is greater than zero, the cacher tracks managed keys in Redis and evicts least recently used entries as needed.

**Parameters:**
- `client`: A `*redis.Client` instance from `github.com/redis/go-redis/v9`
- `maxItems`: Maximum cached items before LRU eviction (`<= 0` means unlimited)

**Returns:**
- A `Cacher[T]` implementation that uses Redis for storage, LRU eviction, and distributed locking

### NewMemoryCacher Function

```go
func NewMemoryCacher[T any](defaultExpiration, cleanupInterval time.Duration, maxItems int) Cacher[T]
```

Creates a new in-memory cacher instance. The type parameter `T` determines what type of values will be cached. When `maxItems` is greater than zero, the cacher evicts least recently used entries as needed.

**Parameters:**
- `defaultExpiration`: Default TTL for cached items (use `cache.NoExpiration` for no default expiration)
- `cleanupInterval`: Interval at which expired items are removed from the cache
- `maxItems`: Maximum cached items before LRU eviction (`<= 0` means unlimited)

**Returns:**
- A `Cacher[T]` implementation that uses in-memory storage, LRU eviction, and singleflight for cache stampede prevention

## Error Handling

### Common Error Scenarios

1. **Fetch Function Errors**: Errors from your fetch function are returned as-is:

```go
fetchFn := func(ctx context.Context) (User, error) {
    return User{}, errors.New("user not found")
}

user, err := cacher.GetOrFetch(ctx, "user:1", ttl, fetchFn)
if err != nil {
    // err will be "user not found"
}
```

2. **Context Cancellation**: If context is cancelled or times out:

```go
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()

user, err := cacher.GetOrFetch(ctx, "user:1", ttl, fetchFn)
if err == context.DeadlineExceeded {
    // Operation timed out
} else if err == context.Canceled {
    // Operation was cancelled
}
```

3. **Redis Connection Errors**: Errors from Redis operations are wrapped:

```go
user, err := cacher.GetOrFetch(ctx, "user:1", ttl, fetchFn)
if err != nil {
    // May contain Redis connection errors, unmarshaling errors, etc.
    log.Printf("Error: %v", err)
}
```

4. **Lock Acquisition Timeout**: If waiting for cache times out (when another goroutine is fetching):

```go
// This happens internally - waiting goroutines will timeout after 30 seconds
// if the fetching goroutine doesn't complete
```

## Performance Considerations

### Cache Hit Performance

- **Latency**: Typically < 1ms (Redis round-trip + unmarshaling)
- **Throughput**: Limited by Redis connection pool and network

### Cache Miss Performance

- **First Request**: Latency = fetch function duration + Redis write
- **Concurrent Requests**: Latency = wait time + Redis read (typically much faster than fetch)

### Memory Usage

- Values are stored as JSON strings in Redis
- Consider data size when choosing TTL values
- Monitor Redis memory usage

### Network Considerations

- All operations require Redis round-trips
- Consider Redis connection pooling for high-throughput scenarios
- Use appropriate timeouts to prevent hanging operations

## Limitations

1. **JSON Serialization**: Values must be JSON-serializable
2. **Redis Dependency**: Requires Redis to be available (for Redis cacher)
3. **Lock Timeout**: Maximum wait time for concurrent requests is 30 seconds
4. **Clear Operation**: `Clear()` removes all keys in the current Redis database - use with caution in production
5. **No Batch Operations**: Each key must be fetched individually
6. **DeleteByPrefix Performance**: For very large key sets, `DeleteByPrefix` may take time as it uses SCAN to find matching keys

## Troubleshooting

### Issue: Cache values not being retrieved

**Possible Causes:**
- Redis connection issues
- JSON unmarshaling errors
- Key mismatch

**Solution:**
```go
// Test Redis connection
if err := redisClient.Ping(ctx).Err(); err != nil {
    log.Fatalf("Redis connection failed: %v", err)
}

// Check key format consistency
key := fmt.Sprintf("user:%d", userID) // Ensure consistent format
```

### Issue: Fetch function called multiple times

**Possible Causes:**
- Lock not being acquired properly
- Lock expiration before fetch completes

**Solution:**
- Ensure Redis is accessible and responsive
- Check that fetch function completes within reasonable time
- Lock is automatically extended, but very long operations may still timeout

### Issue: High memory usage in Redis

**Possible Causes:**
- TTL values too long
- Large cached values
- Too many cached keys

**Solution:**
- Reduce TTL values for less critical data
- Consider compressing large values before caching
- Implement cache eviction policies
