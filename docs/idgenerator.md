# IdGenerator Documentation

The `idgenerator` package provides a concurrency-safe generator of monotonically increasing `uint32` IDs. Each call to `Id()` returns the next ID by atomically incrementing an internal counter. The starting value is set at construction; the first call to `Id()` returns `startValue + 1`. It is suitable for session IDs, request IDs, or any per-entity identifier within a single process.

## Features

- **Monotonic**: IDs always increase; no duplicates for sequential calls
- **Concurrent safe**: Uses `atomic.Uint32`; safe for use from multiple goroutines
- **Simple API**: `NewIdGenerator(startValue)` and `Id()` only
- **No dependencies**: Only uses the standard library `sync/atomic`

## Installation

```go
import "github.com/cyberinferno/go-utils/idgenerator"
```

## Creating a Generator

### NewIdGenerator

Creates an `IdGenerator` that will generate IDs starting from `startValue + 1`. The generator is safe for concurrent use.

```go
gen := idgenerator.NewIdGenerator(0)
first := gen.Id()  // 1
second := gen.Id() // 2
third := gen.Id()  // 3
```

Starting from a non-zero value:

```go
gen := idgenerator.NewIdGenerator(1000)
id := gen.Id() // 1001
id = gen.Id()  // 1002
```

**Parameters:**

- **startValue**: The value to initialize the internal counter to. The first call to `Id()` returns `startValue + 1`.

**Returns:**

- A new `*IdGenerator` instance.

---

## Getting the Next ID

### Id

Returns the next unique ID by atomically incrementing the internal counter. Safe for concurrent use by multiple goroutines.

```go
id := gen.Id()
```

**Returns:**

- The next `uint32` ID.

Multiple goroutines can call `Id()` simultaneously; each call returns a distinct value (assuming no overflow; see Limitations).

```go
var wg sync.WaitGroup
gen := idgenerator.NewIdGenerator(0)
ids := make([]uint32, 100)

for i := range 100 {
	wg.Add(1)
	go func(i int) {
		defer wg.Done()
		ids[i] = gen.Id()
	}(i)
}
wg.Wait()
// ids contains 100 unique values (e.g. 1..100 in some order)
```

---

## Basic Usage

### Session or connection IDs

Common use case is to assign a unique ID to each TCP session or connection:

```go
gen := idgenerator.NewIdGenerator(0)

// In your TCP server accept loop:
id := gen.Id()
session := NewSession(id, conn)
server.AddSession(id, session)
```

### Request or trace IDs

Use for request IDs, trace IDs, or log correlation:

```go
requestID := idGen.Id()
log.Printf("[%d] request started", requestID)
```

### Starting from a reserved range

Reserve 0 for “invalid” or “broadcast” and start IDs from 1:

```go
gen := idgenerator.NewIdGenerator(0)
// First Id() is 1; 0 can mean "no session" or "broadcast"
```

Or start from a high base to avoid collision with other ID spaces:

```go
gen := idgenerator.NewIdGenerator(1_000_000)
```

---

## Complete Examples

### Example 1: Single goroutine

```go
package main

import (
	"fmt"
	"github.com/cyberinferno/go-utils/idgenerator"
)

func main() {
	gen := idgenerator.NewIdGenerator(0)
	for i := 0; i < 5; i++ {
		fmt.Println(gen.Id())
	}
	// Output (order): 1, 2, 3, 4, 5
}
```

### Example 2: Concurrent IDs

```go
package main

import (
	"fmt"
	"sync"
	"github.com/cyberinferno/go-utils/idgenerator"
)

func main() {
	gen := idgenerator.NewIdGenerator(0)
	var wg sync.WaitGroup
	mu := sync.Mutex{}
	var seen []uint32

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := gen.Id()
			mu.Lock()
			seen = append(seen, id)
			mu.Unlock()
		}()
	}
	wg.Wait()
	fmt.Println(seen) // 10 unique IDs
}
```

### Example 3: With TCPServer

Used by the `tcpserver` package to assign session IDs:

```go
srv := &tcpserver.TCPServer{
	// ...
	IdGenerator: idgenerator.NewIdGenerator(0),
	NewSession: func(id uint32, conn net.Conn) tcpserver.TCPServerSession {
		return &MySession{id: id, conn: conn}
	},
}
```

---

## Type Reference

### IdGenerator

```go
type IdGenerator struct {
	// start and id are unexported; use NewIdGenerator and Id()
}
```

Generates monotonically increasing `uint32` IDs in a concurrency-safe manner. Do not copy; use a pointer and pass it where needed.

### NewIdGenerator

```go
func NewIdGenerator(startValue uint32) *IdGenerator
```

Creates an `IdGenerator` that will return `startValue+1`, `startValue+2`, ... from subsequent `Id()` calls.

**Parameters:**

- **startValue**: Initial counter value; first `Id()` returns `startValue + 1`.

**Returns:**

- A new `*IdGenerator` instance.

### Id

```go
func (l *IdGenerator) Id() uint32
```

Returns the next unique ID. Safe for concurrent use.

**Returns:**

- The next `uint32` ID.

---

## Best Practices

1. **Single generator per ID space**: Use one `IdGenerator` per logical ID space (e.g. one for TCP sessions, one for requests) so IDs are unique within that space.

2. **Pass by pointer**: Pass `*IdGenerator` to servers or handlers; do not copy the struct (same as other sync/atomic-based types).

3. **Reserve zero if needed**: Use `NewIdGenerator(0)` so the first ID is 1 and reserve 0 for “none” or “broadcast” in your protocol or storage.

---

## Limitations

- **uint32 overflow**: After 4,294,967,295 IDs, the counter wraps to 0. For long-lived processes that might exceed this, consider a larger type or a different ID scheme.
- **No persistence**: The counter is in-memory only. Restarting the process resets the sequence; if you need globally unique or restart-safe IDs, use a different mechanism (e.g. UUID, database sequence).
- **Single process**: IDs are unique only within one process. For distributed systems, combine with a node/shard ID or use another source of uniqueness.
