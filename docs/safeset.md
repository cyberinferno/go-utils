# Safeset Documentation

The `safeset` package provides a thread-safe set that stores a collection of unique elements of comparable type. It is built on a map and `sync.RWMutex`, and supports add, remove, membership test, size, iteration, and set operations (intersection and union). SafeSet is safe for use by multiple goroutines without additional locking.

## Features

- **Type-Safe Generics**: Elements are generic with type parameter `T comparable`; no `interface{}` casting at the call site
- **Concurrent Safe**: Uses `sync.RWMutex`; safe for concurrent reads and writes from multiple goroutines
- **Set Semantics**: Uniqueness of elements; duplicate adds do not increase size
- **Set Operations**: Intersection and Union return new sets; original sets are unchanged
- **Familiar API**: Add, Remove, Contains, Size, Reset, and Range mirror common set operations
- **O(1) Size**: Number of elements is maintained by the underlying map; Size is O(1)

## Installation

```go
import "github.com/cyberinferno/go-utils/safeset"
```

## Creating a SafeSet

### NewSafeSet

Creates a new, empty SafeSet. The type parameter is the element type `T`, which must be [comparable](https://go.dev/ref/spec#Comparison_operators).

```go
import "github.com/cyberinferno/go-utils/safeset"

// Set of strings
s := safeset.NewSafeSet[string]()

// Set of integers
s := safeset.NewSafeSet[int]()

// Set of custom comparable type (e.g. struct with comparable fields)
type Key struct{ A, B int }
s := safeset.NewSafeSet[Key]()
```

**Returns:**

- A pointer to a new `SafeSet[T]` that is empty and safe for concurrent use.

---

## Basic Usage

### Add

Adds an element to the set. If the element is already present, the set is unchanged (no duplicate entries).

```go
s := safeset.NewSafeSet[string]()
s.Add("a")
s.Add("b")
s.Add("a")  // no-op for duplicate
// s.Size() == 2
```

**Parameters:**

- **value**: The element to add

---

### Remove

Removes an element from the set. Safe to call for an element that is not in the set (no-op).

```go
s.Remove("a")
```

**Parameters:**

- **value**: The element to remove

---

### Contains

Reports whether the set contains the given element.

```go
s := safeset.NewSafeSet[int]()
s.Add(42)

if s.Contains(42) {
    // element is in the set
}

if !s.Contains(99) {
    // element is not in the set
}
```

**Parameters:**

- **value**: The element to look up

**Returns:**

- `true` if the set contains the element, `false` otherwise

---

### Size

Returns the number of elements in the set. This is O(1) because it uses the length of the underlying map.

```go
n := s.Size()
```

**Returns:**

- The number of elements in the set

---

### Reset

Removes all elements from the set, leaving it empty. The set can be used again after Reset.

```go
s.Add(1)
s.Add(2)
s.Reset()
// s.Size() == 0, s.Contains(1) == false
```

---

### Range

Calls a function for each element in the set. If the function returns `false`, iteration stops. Do not modify the set from within the callback; behavior is undefined if you do.

```go
s := safeset.NewSafeSet[string]()
s.Add("x")
s.Add("y")
s.Add("z")

s.Range(func(v string) bool {
    fmt.Println(v)
    return true  // continue; return false to stop
})
```

**Parameters:**

- **f**: Function called for each element; return `false` to stop iteration

---

## Set Operations

### Intersection

Returns a new set containing only the elements that are present in both this set and the other set. The receiver and `other` are not modified.

```go
a := safeset.NewSafeSet[int]()
a.Add(1)
a.Add(2)
a.Add(3)

b := safeset.NewSafeSet[int]()
b.Add(2)
b.Add(3)
b.Add(4)

inter := a.Intersection(b)
// inter contains 2 and 3; Size() == 2
```

**Parameters:**

- **other**: The other set to intersect with

**Returns:**

- A new `*SafeSet[T]` containing the intersection of the two sets

---

### Union

Returns a new set containing all elements that are in this set, the other set, or both. The receiver and `other` are not modified.

```go
a := safeset.NewSafeSet[int]()
a.Add(1)
a.Add(2)

b := safeset.NewSafeSet[int]()
b.Add(2)
b.Add(3)

u := a.Union(b)
// u contains 1, 2, 3; Size() == 3
```

**Parameters:**

- **other**: The other set to union with

**Returns:**

- A new `*SafeSet[T]` containing the union of the two sets

---

## Element Type

Elements must be [comparable](https://go.dev/ref/spec#Comparison_operators). Common choices:

- Basic types: `string`, `int`, `int64`, `float64`, etc.
- Pointers
- Structs whose fields are all comparable (no slices, maps, or funcs)

Slices and maps are not comparable and cannot be used as set elements.

```go
safeset.NewSafeSet[string]()
safeset.NewSafeSet[int]()
safeset.NewSafeSet[*MyStruct]()
```

---

## Concurrency

SafeSet is safe for concurrent use. Multiple goroutines may call Add, Remove, Contains, Size, Reset, Range, Intersection, and Union simultaneously. Range may run concurrently with other operations; do not add or remove elements from inside the Range callback.

```go
var wg sync.WaitGroup
s := safeset.NewSafeSet[int]()

for i := range 100 {
    wg.Add(1)
    go func(v int) {
        defer wg.Done()
        s.Add(v)
    }(i)
}
wg.Wait()

fmt.Println(s.Size()) // 100
```

---

## Complete Examples

### Example 1: Unique Collection

```go
package main

import (
    "fmt"
    "github.com/cyberinferno/go-utils/safeset"
)

func main() {
    seen := safeset.NewSafeSet[string]()
    for _, name := range []string{"alice", "bob", "alice", "carol", "bob"} {
        seen.Add(name)
    }
    fmt.Println("unique names:", seen.Size())

    seen.Range(func(v string) bool {
        fmt.Println(v)
        return true
    })
}
```

### Example 2: Allowed IDs

```go
package main

import (
    "fmt"
    "github.com/cyberinferno/go-utils/safeset"
)

func main() {
    allowed := safeset.NewSafeSet[int]()
    allowed.Add(1)
    allowed.Add(2)
    allowed.Add(3)

    for _, id := range []int{1, 5, 2, 9} {
        if allowed.Contains(id) {
            fmt.Println(id, "is allowed")
        } else {
            fmt.Println(id, "is not allowed")
        }
    }
}
```

### Example 3: Intersection and Union

```go
package main

import (
    "fmt"
    "github.com/cyberinferno/go-utils/safeset"
)

func main() {
    admins := safeset.NewSafeSet[string]()
    admins.Add("alice")
    admins.Add("bob")
    admins.Add("carol")

    active := safeset.NewSafeSet[string]()
    active.Add("bob")
    active.Add("carol")
    active.Add("dave")

    activeAdmins := admins.Intersection(active)
    fmt.Println("active admins:", activeAdmins.Size())
    activeAdmins.Range(func(v string) bool {
        fmt.Println(v)
        return true
    })

    all := admins.Union(active)
    fmt.Println("all (admins + active):", all.Size())
}
```

### Example 4: Early Exit from Range

```go
s := safeset.NewSafeSet[int]()
// ... populate s ...

var found bool
s.Range(func(v int) bool {
    if v == 100 {
        found = true
        return false // stop iteration
    }
    return true
})
```

### Example 5: Concurrent Deduplication

```go
package main

import (
    "sync"
    "github.com/cyberinferno/go-utils/safeset"
)

func main() {
    ids := safeset.NewSafeSet[int64]()
    var wg sync.WaitGroup

    for _, id := range []int64{1, 2, 1, 3, 2, 4, 1} {
        wg.Add(1)
        go func(v int64) {
            defer wg.Done()
            ids.Add(v)
        }(id)
    }
    wg.Wait()

    // ids.Size() is 4 (unique: 1, 2, 3, 4)
}
```

---

## Type Reference

### SafeSet

```go
type SafeSet[T comparable] struct {
    // contains map[T]struct{} and sync.RWMutex (unexported)
}
```

Thread-safe set of unique elements of type T. T must be comparable. Safe for concurrent use. Do not copy after first use.

### NewSafeSet

```go
func NewSafeSet[T comparable]() *SafeSet[T]
```

Returns a new empty SafeSet.

### Methods

| Method | Description |
|--------|-------------|
| `Add(value T)` | Adds an element to the set (no-op if already present). |
| `Remove(value T)` | Removes an element; no-op if not present. |
| `Contains(value T) bool` | Reports whether the set contains the element. |
| `Size() int` | Returns the number of elements (O(1)). |
| `Reset()` | Removes all elements from the set. |
| `Range(f func(value T) bool)` | Calls `f` for each element; stop by returning false. |
| `Intersection(other *SafeSet[T]) *SafeSet[T]` | Returns a new set with elements in both sets. |
| `Union(other *SafeSet[T]) *SafeSet[T]` | Returns a new set with elements in either set. |

---

## Best Practices

1. **Use Contains for membership**: For a simple “is this in the set?” check, use `Contains(value)`.

2. **Avoid modifying inside Range**: Do not Add or Remove from within the Range callback; behavior is undefined.

3. **Set operations allocate**: Intersection and Union create new sets; reuse or discard as needed to manage memory.

4. **Prefer comparable element types**: Use simple types (string, int) or structs with comparable fields for clarity and correctness.

5. **Nil other**: Intersection and Union assume `other` is non-nil; passing nil will panic when ranging over `other.m`.

---

## Limitations

- **No copy**: The set must not be copied after first use (same as types containing mutexes).
- **Elements must be comparable**: Slices, maps, and non-comparable structs cannot be used as elements.
- **No snapshot**: Range may see concurrent mutations; it does not iterate a snapshot.
- **Nil other**: For Intersection and Union, the `other` argument must not be nil.
