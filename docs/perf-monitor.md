# Performance Monitor Documentation

The `perfmonitor` package provides a simple and efficient way to measure elapsed time between operations in milliseconds. It's designed for performance monitoring, benchmarking, and timing operations in Go applications.

## Features

- **Simple API**: Easy-to-use Start/Stop pattern for timing operations
- **Millisecond Precision**: Returns elapsed time in milliseconds with microsecond precision
- **Safe Operations**: Handles edge cases like stopping without starting
- **Reusable**: Reset functionality allows reuse of the same monitor instance
- **Zero Overhead**: Lightweight implementation with minimal memory footprint

## Installation

```go
import "github.com/cyberinferno/go-utils/perfmonitor"
```

## Creating a Performance Monitor

Create a new performance monitor instance using `NewPerformanceMonitor`:

```go
import "github.com/cyberinferno/go-utils/perfmonitor"

pm := perfmonitor.NewPerformanceMonitor()
```

The monitor is initialized with zero times and ready to use immediately.

## Basic Usage

### Measuring Operation Duration

The most common use case is to measure how long an operation takes:

```go
pm := perfmonitor.NewPerformanceMonitor()

// Start timing
pm.Start()

// Perform some operation
doSomeWork()

// Stop timing
pm.Stop()

// Get elapsed time in milliseconds
elapsed := pm.ElapsedMilliseconds()
fmt.Printf("Operation took %.2f ms\n", elapsed)
```

### Complete Example

```go
package main

import (
    "fmt"
    "time"
    "github.com/cyberinferno/go-utils/perfmonitor"
)

func main() {
    pm := perfmonitor.NewPerformanceMonitor()
    
    pm.Start()
    time.Sleep(100 * time.Millisecond) // Simulate work
    pm.Stop()
    
    elapsed := pm.ElapsedMilliseconds()
    fmt.Printf("Operation completed in %.2f ms\n", elapsed)
    // Output: Operation completed in ~100.00 ms
}
```

## API Reference

### NewPerformanceMonitor

Creates a new `PerformanceMonitor` instance.

```go
func NewPerformanceMonitor() *PerformanceMonitor
```

**Returns**: A new `PerformanceMonitor` instance with zero start and end times.

### Start

Begins the performance monitoring by recording the current time.

```go
func (p *PerformanceMonitor) Start()
```

**Behavior**:
- Sets the start time to the current time
- Can be called multiple times (overwrites previous start time)
- Does not affect the end time

### Stop

Ends the performance monitoring by recording the current time.

```go
func (p *PerformanceMonitor) Stop()
```

**Behavior**:
- Sets the end time to the current time
- Only works if `Start()` was called first
- If `Start()` was not called, the method does nothing
- Can be called multiple times (overwrites previous end time)

### ElapsedMilliseconds

Returns the elapsed time in milliseconds between `Start()` and `Stop()`.

```go
func (p *PerformanceMonitor) ElapsedMilliseconds() float64
```

**Returns**: 
- Elapsed time in milliseconds as a `float64`
- Returns `0.0` if `Start()` was not called
- Returns `0.0` if `Stop()` was not called
- Returns `0.0` if both times are zero (after `Reset()`)

**Precision**: The function uses microseconds internally and converts to milliseconds, providing sub-millisecond precision.

### Reset

Clears both start and end times, allowing the monitor to be reused.

```go
func (p *PerformanceMonitor) Reset()
```

**Behavior**:
- Sets both start and end times to zero
- Safe to call multiple times
- After reset, `ElapsedMilliseconds()` returns `0.0` until `Start()` and `Stop()` are called again

## Usage Examples

### Measuring Function Execution Time

```go
func processData(data []int) error {
    pm := perfmonitor.NewPerformanceMonitor()
    
    pm.Start()
    defer pm.Stop()
    
    // Process data
    for _, item := range data {
        // ... processing logic
    }
    
    elapsed := pm.ElapsedMilliseconds()
    fmt.Printf("Processed %d items in %.2f ms\n", len(data), elapsed)
    
    return nil
}
```

### Measuring Database Query Time

```go
func fetchUser(userID string) (*User, error) {
    pm := perfmonitor.NewPerformanceMonitor()
    
    pm.Start()
    user, err := db.Query("SELECT * FROM users WHERE id = ?", userID)
    pm.Stop()
    
    if err != nil {
        return nil, err
    }
    
    elapsed := pm.ElapsedMilliseconds()
    log.Printf("Database query took %.2f ms", elapsed)
    
    return user, nil
}
```

### Measuring HTTP Request Duration

```go
func makeAPICall(url string) (*Response, error) {
    pm := perfmonitor.NewPerformanceMonitor()
    
    pm.Start()
    resp, err := http.Get(url)
    pm.Stop()
    
    elapsed := pm.ElapsedMilliseconds()
    
    if err != nil {
        log.Printf("API call failed after %.2f ms: %v", elapsed, err)
        return nil, err
    }
    
    log.Printf("API call completed in %.2f ms", elapsed)
    return resp, nil
}
```

### Reusing a Monitor

The monitor can be reset and reused for multiple measurements:

```go
pm := perfmonitor.NewPerformanceMonitor()

// First measurement
pm.Start()
operation1()
pm.Stop()
elapsed1 := pm.ElapsedMilliseconds()

// Reset and reuse
pm.Reset()

// Second measurement
pm.Start()
operation2()
pm.Stop()
elapsed2 := pm.ElapsedMilliseconds()

fmt.Printf("Operation 1: %.2f ms, Operation 2: %.2f ms\n", elapsed1, elapsed2)
```

### Measuring Multiple Operations

You can measure multiple operations without resetting:

```go
pm := perfmonitor.NewPerformanceMonitor()

// Measure first operation
pm.Start()
doFirstOperation()
pm.Stop()
firstElapsed := pm.ElapsedMilliseconds()

// Measure second operation (reuses same monitor)
pm.Start()
doSecondOperation()
pm.Stop()
secondElapsed := pm.ElapsedMilliseconds()

fmt.Printf("First: %.2f ms, Second: %.2f ms\n", firstElapsed, secondElapsed)
```

### Performance Logging with Context

Combine with logging to track performance:

```go
func handleRequest(ctx context.Context, req *Request) (*Response, error) {
    pm := perfmonitor.NewPerformanceMonitor()
    
    pm.Start()
    defer func() {
        pm.Stop()
        elapsed := pm.ElapsedMilliseconds()
        log.Printf("Request handled in %.2f ms", elapsed)
    }()
    
    // Process request
    return processRequest(ctx, req)
}
```

### Benchmarking Multiple Functions

Compare performance of different implementations:

```go
func benchmarkFunctions() {
    pm := perfmonitor.NewPerformanceMonitor()
    
    // Benchmark function A
    pm.Start()
    resultA := functionA()
    pm.Stop()
    elapsedA := pm.ElapsedMilliseconds()
    
    // Benchmark function B
    pm.Reset()
    pm.Start()
    resultB := functionB()
    pm.Stop()
    elapsedB := pm.ElapsedMilliseconds()
    
    fmt.Printf("Function A: %.2f ms, Function B: %.2f ms\n", elapsedA, elapsedB)
    fmt.Printf("Winner: %s\n", getWinner(elapsedA, elapsedB))
}
```

## Edge Cases and Safety

### Stop Without Start

If `Stop()` is called without calling `Start()` first, it does nothing:

```go
pm := perfmonitor.NewPerformanceMonitor()

pm.Stop() // Does nothing, no error

elapsed := pm.ElapsedMilliseconds()
fmt.Println(elapsed) // Output: 0
```

### Elapsed Without Start or Stop

If `ElapsedMilliseconds()` is called before `Start()` or `Stop()`, it returns `0.0`:

```go
pm := perfmonitor.NewPerformanceMonitor()

// Before Start
elapsed := pm.ElapsedMilliseconds()
fmt.Println(elapsed) // Output: 0

pm.Start()
elapsed = pm.ElapsedMilliseconds()
fmt.Println(elapsed) // Output: 0 (Stop not called yet)

pm.Stop()
elapsed = pm.ElapsedMilliseconds()
fmt.Println(elapsed) // Output: actual elapsed time
```

### Multiple Start Calls

Calling `Start()` multiple times overwrites the previous start time:

```go
pm := perfmonitor.NewPerformanceMonitor()

pm.Start()
time.Sleep(10 * time.Millisecond)
pm.Start() // Resets start time
time.Sleep(10 * time.Millisecond)
pm.Stop()

elapsed := pm.ElapsedMilliseconds()
// Only measures time from second Start() to Stop()
fmt.Printf("Elapsed: %.2f ms\n", elapsed) // ~10 ms, not ~20 ms
```

### Multiple Stop Calls

Calling `Stop()` multiple times updates the end time:

```go
pm := perfmonitor.NewPerformanceMonitor()

pm.Start()
time.Sleep(10 * time.Millisecond)
pm.Stop()
firstElapsed := pm.ElapsedMilliseconds()

time.Sleep(10 * time.Millisecond)
pm.Stop() // Updates end time
secondElapsed := pm.ElapsedMilliseconds()

// secondElapsed will be greater than firstElapsed
fmt.Printf("First: %.2f ms, Second: %.2f ms\n", firstElapsed, secondElapsed)
```

## Common Use Cases

### API Endpoint Performance Monitoring

```go
func apiHandler(w http.ResponseWriter, r *http.Request) {
    pm := perfmonitor.NewPerformanceMonitor()
    
    pm.Start()
    defer func() {
        pm.Stop()
        elapsed := pm.ElapsedMilliseconds()
        
        // Log performance metrics
        log.WithFields(log.Fields{
            "endpoint": r.URL.Path,
            "method": r.Method,
            "duration_ms": elapsed,
        }).Info("Request completed")
    }()
    
    // Handle request
    handleRequest(w, r)
}
```

### Database Operation Timing

```go
type DBService struct {
    db *sql.DB
}

func (s *DBService) QueryUsers() ([]User, error) {
    pm := perfmonitor.NewPerformanceMonitor()
    
    pm.Start()
    rows, err := s.db.Query("SELECT * FROM users")
    pm.Stop()
    
    elapsed := pm.ElapsedMilliseconds()
    
    if err != nil {
        log.Printf("Query failed after %.2f ms: %v", elapsed, err)
        return nil, err
    }
    
    log.Printf("Query completed in %.2f ms", elapsed)
    defer rows.Close()
    
    // Process rows...
    return users, nil
}
```

### Cache Operation Timing

```go
func getCachedData(key string) ([]byte, error) {
    pm := perfmonitor.NewPerformanceMonitor()
    
    pm.Start()
    data, err := cache.Get(key)
    pm.Stop()
    
    elapsed := pm.ElapsedMilliseconds()
    
    if err == nil {
        log.Printf("Cache hit for key '%s' in %.2f ms", key, elapsed)
    } else {
        log.Printf("Cache miss for key '%s' (checked in %.2f ms)", key, elapsed)
    }
    
    return data, err
}
```

### Batch Processing Performance

```go
func processBatch(items []Item) error {
    pm := perfmonitor.NewPerformanceMonitor()
    
    pm.Start()
    for i, item := range items {
        if err := processItem(item); err != nil {
            return err
        }
        
        // Log progress every 100 items
        if (i+1)%100 == 0 {
            pm.Stop()
            elapsed := pm.ElapsedMilliseconds()
            rate := float64(i+1) / (elapsed / 1000.0) // items per second
            log.Printf("Processed %d/%d items (%.2f items/sec)", i+1, len(items), rate)
            pm.Start() // Continue timing
        }
    }
    pm.Stop()
    
    totalElapsed := pm.ElapsedMilliseconds()
    log.Printf("Batch processing completed in %.2f ms", totalElapsed)
    
    return nil
}
```

### Performance Comparison

```go
func compareAlgorithms(data []int) {
    pm := perfmonitor.NewPerformanceMonitor()
    
    // Test algorithm A
    pm.Start()
    resultA := algorithmA(data)
    pm.Stop()
    elapsedA := pm.ElapsedMilliseconds()
    
    // Test algorithm B
    pm.Reset()
    pm.Start()
    resultB := algorithmB(data)
    pm.Stop()
    elapsedB := pm.ElapsedMilliseconds()
    
    fmt.Printf("Algorithm A: %.2f ms\n", elapsedA)
    fmt.Printf("Algorithm B: %.2f ms\n", elapsedB)
    fmt.Printf("Speedup: %.2fx\n", elapsedA/elapsedB)
    
    // Verify results are the same
    if !resultsEqual(resultA, resultB) {
        log.Fatal("Algorithms produced different results!")
    }
}
```

## Best Practices

1. **Use Defer for Cleanup**: When measuring function execution time, use `defer` to ensure `Stop()` is always called:

   ```go
   func myFunction() {
       pm := perfmonitor.NewPerformanceMonitor()
       pm.Start()
       defer pm.Stop()
       
       // Your code here
   }
   ```

2. **Reset Before Reuse**: If reusing a monitor instance, always call `Reset()` between measurements to ensure clean state:

   ```go
   pm.Reset()
   pm.Start()
   // ... operation
   pm.Stop()
   ```

3. **Log Performance Metrics**: Include elapsed time in your logs for monitoring and debugging:

   ```go
   elapsed := pm.ElapsedMilliseconds()
   log.Printf("Operation completed in %.2f ms", elapsed)
   ```

4. **Handle Zero Values**: Always check if elapsed time is greater than zero before using it:

   ```go
   elapsed := pm.ElapsedMilliseconds()
   if elapsed > 0 {
       // Use elapsed time
   }
   ```

5. **Use for Critical Paths**: Focus on measuring performance of critical operations, not every function call.

6. **Combine with Logging**: Use performance monitoring alongside structured logging for better observability.

7. **Monitor in Production**: Use performance monitoring in production to identify slow operations, but be mindful of overhead.

8. **Set Performance Thresholds**: Define acceptable performance thresholds and alert when operations exceed them.

## Type Reference

### PerformanceMonitor

```go
type PerformanceMonitor struct {
    startTime time.Time
    endTime   time.Time
}
```

The `PerformanceMonitor` struct contains the start and end times for measuring elapsed duration. Fields are unexported and should be accessed through the provided methods.

### Methods

```go
// Start begins the performance monitoring
func (p *PerformanceMonitor) Start()

// Stop ends the performance monitoring
func (p *PerformanceMonitor) Stop()

// ElapsedMilliseconds returns the elapsed time in milliseconds
func (p *PerformanceMonitor) ElapsedMilliseconds() float64

// Reset clears the timer values to allow reuse
func (p *PerformanceMonitor) Reset()
```

### Constructor

```go
// NewPerformanceMonitor creates a new PerformanceMonitor instance
func NewPerformanceMonitor() *PerformanceMonitor
```

## Notes

- **Precision**: The monitor uses `time.Now()` internally, which provides nanosecond precision. The `ElapsedMilliseconds()` method converts microseconds to milliseconds, providing sub-millisecond precision in the result.

- **Thread Safety**: The `PerformanceMonitor` is not thread-safe. If you need to use it from multiple goroutines, you should use synchronization primitives or create separate monitor instances for each goroutine.

- **Overhead**: The overhead of the monitor itself is minimal (a few nanoseconds per operation), making it suitable for production use.

- **Time Source**: The monitor uses `time.Now()` which uses the system clock. Clock adjustments (e.g., NTP corrections) may affect measurements over long durations.

- **Zero Time Handling**: The monitor uses `time.Time.IsZero()` to check if times are set. This is a safe way to handle uninitialized states.
