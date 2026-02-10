# Safemap Documentation

The `safemap` package provides a type-safe, concurrent map built on Go's `sync.Map`. It supports arbitrary comparable keys and any value type, with a consistent API for storing, loading, deleting, and iterating entries. SafeMap is safe for use by multiple goroutines without additional locking.

## Features

- **Type-Safe Generics**: Keys and values are generic (`K comparable`, `V any`); no `interface{}` casting at the call site
- **Concurrent Safe**: Wraps `sync.Map`; safe for concurrent reads and writes from multiple goroutines
- **Familiar API**: Store/Load (or Set/Get), Delete, Range, Len, Has mirror common map operations
- **Zero-Value Safe**: Load of a missing key returns the zero value for V and `false`; no panics
- **No Copy After Use**: Like `sync.Map`, the map must not be copied after first use

## Installation

```go
import "github.com/cyberinferno/go-utils/safemap"
```

## Creating a SafeMap

### NewSafeMap

Creates a new, empty SafeMap. The type parameters are the key type `K` (must be comparable) and the value type `V` (any).

```go
import "github.com/cyberinferno/go-utils/safemap"

// String keys, int values
m := safemap.NewSafeMap[string, int]()

// Int keys, struct values
m := safemap.NewSafeMap[int, struct{ Name string }]()

// Custom comparable key (e.g. struct with comparable fields)
type Key struct{ A, B int }
m := safemap.NewSafeMap[Key, string]()
```

**Returns:**

- A pointer to a new `SafeMap[K, V]` that is empty and safe for concurrent use.

---

## Basic Usage

### Store and Set

Store or set a value for a key. Both overwrite any existing value for that key.

```go
m := safemap.NewSafeMap[string, int]()
m.Store("a", 1)
m.Set("b", 2)  // Set is equivalent to Store
```

**Parameters:**

- **k**: The key to store or set
- **v**: The value to associate with the key

---

### Load and Get

Retrieve a value by key. Both return the value and a boolean indicating whether the key was present. If the key is missing, the value is the zero value for `V` and the boolean is `false`.

```go
m := safemap.NewSafeMap[string, int]()
m.Store("x", 42)

v, ok := m.Load("x")
// v == 42, ok == true

v, ok = m.Load("missing")
// v == 0, ok == false
```

**Parameters:**

- **k**: The key to look up

**Returns:**

- The value associated with the key, or the zero value of `V` if not found
- `true` if the key was present, `false` otherwise

---

### Delete

Removes the entry for a key. Safe to call for a key that is not in the map (no-op).

```go
m.Delete("x")
```

**Parameters:**

- **k**: The key to delete

---

### Has

Reports whether a key is present in the map.

```go
if m.Has("user:123") {
    // key exists
}
```

**Parameters:**

- **k**: The key to check

**Returns:**

- `true` if the key is in the map, `false` otherwise

---

### Len

Returns the number of entries in the map. Implemented by iterating over all entries, so use sparingly on large maps.

```go
n := m.Len()
```

**Returns:**

- The number of key-value pairs in the map

---

### Range

Calls a function for each key-value pair in the map. If the function returns `false`, iteration stops. Do not modify the map from within the callback; behavior is undefined if you do.

```go
m.Range(func(k string, v int) bool {
    fmt.Println(k, v)
    return true  // continue; return false to stop
})
```

**Parameters:**

- **f**: Function called for each entry; return `false` to stop iteration.

---

## Key and Value Types

- **Keys**: Must be [comparable](https://go.dev/ref/spec#Comparison_operators) (e.g. `string`, `int`, pointers, structs of comparable fields). Slices and maps are not comparable and cannot be used as keys.
- **Values**: Any type `V` is allowed, including pointers, slices, structs, and other maps.

```go
// Common key/value combinations
safemap.NewSafeMap[string, int]()
safemap.NewSafeMap[int, *MyStruct]()
safemap.NewSafeMap[string, []byte]()
```

---

## Concurrency

SafeMap is safe for concurrent use. Multiple goroutines may call Store, Load, Set, Get, Delete, Has, Len, and Range simultaneously. Range may run concurrently with other operations; do not add or delete keys from inside the Range callback.

```go
var wg sync.WaitGroup
m := safemap.NewSafeMap[int, int]()

for i := range 100 {
    wg.Add(1)
    go func(k int) {
        defer wg.Done()
        m.Store(k, k*2)
    }(i)
}
wg.Wait()

fmt.Println(m.Len()) // 100
```

---

## Complete Examples

### Example 1: In-Process Cache

```go
package main

import (
    "fmt"
    "github.com/cyberinferno/go-utils/safemap"
)

func main() {
    cache := safemap.NewSafeMap[string, string]()
    cache.Store("user:1", "Alice")
    cache.Store("user:2", "Bob")

    if v, ok := cache.Get("user:1"); ok {
        fmt.Println("user:1 =>", v)
    }

    cache.Delete("user:2")
    fmt.Println("entries:", cache.Len())
}
```

### Example 2: Set-Like Usage (Keys Only)

Use a map with an empty struct or bool for set-like behavior:

```go
m := safemap.NewSafeMap[int, struct{}]()
m.Store(1, struct{}{})
m.Store(2, struct{}{})

if m.Has(1) {
    fmt.Println("1 is in the set")
}

m.Range(func(k int, _ struct{}) bool {
    fmt.Println("member:", k)
    return true
})
```

### Example 3: Concurrent Counters by Key

```go
package main

import (
    "sync"
    "github.com/cyberinferno/go-utils/safemap"
)

func main() {
    counts := safemap.NewSafeMap[string, int]()
    var wg sync.WaitGroup

    for _, key := range []string{"a", "b", "a", "b", "a"} {
        wg.Add(1)
        go func(k string) {
            defer wg.Done()
            if v, ok := counts.Load(k); ok {
                counts.Store(k, v+1)
            } else {
                counts.Store(k, 1)
            }
        }(key)
    }
    wg.Wait()

    counts.Range(func(k string, v int) bool {
        fmt.Printf("%s: %d\n", k, v)
        return true
    })
}
```

### Example 4: Early Exit from Range

```go
m := safemap.NewSafeMap[string, int]()
// ... populate m ...

var found string
m.Range(func(k string, v int) bool {
    if v == 100 {
        found = k
        return false // stop iteration
    }
    return true
})
```

---

## Type Reference

### SafeMap

```go
type SafeMap[K comparable, V any] struct {
    // contains sync.Map (unexported)
}
```

Concurrent map with type-safe generic API. Keys must be comparable; values may be any type. Do not copy after first use.

### NewSafeMap

```go
func NewSafeMap[K comparable, V any]() *SafeMap[K, V]
```

Returns a new empty SafeMap.

### Methods

| Method   | Description |
|----------|-------------|
| `Store(k K, v V)` | Sets the value for key `k` (overwrites if present). |
| `Set(k K, v V)`   | Same as Store. |
| `Load(k K) (V, bool)` | Returns value and presence for key `k`. |
| `Get(k K) (V, bool)`  | Same as Load. |
| `Delete(k K)`     | Removes key `k`; no-op if not present. |
| `Has(k K) bool`   | Reports whether key `k` is present. |
| `Len() int`       | Returns the number of entries (O(n)). |
| `Range(f func(k K, v V) bool)` | Calls `f` for each entry; stop by returning false. |

---

## Best Practices

1. **Use Load/Get for presence and value**: Prefer `v, ok := m.Load(k)` when you need both the value and whether the key existed; use `Has(k)` when you only need presence.

2. **Avoid modifying inside Range**: Do not Store or Delete from within the Range callback; behavior is undefined.

3. **Use Len sparingly**: Len is O(n); on large maps prefer maintaining a counter or approximate size if you need frequent counts.

4. **Prefer comparable key types**: Use simple types (string, int) or structs with comparable fields as keys for clarity and performance.

---

## Limitations

- **No copy**: The map must not be copied after first use (same as `sync.Map`).
- **Len is O(n)**: Length is computed by iterating all entries.
- **No range snapshot**: Range may see concurrent mutations; it does not iterate a snapshot.
- **Keys must be comparable**: Slices, maps, and non-comparable structs cannot be used as keys.
