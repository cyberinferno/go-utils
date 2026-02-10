package idgenerator

import "sync/atomic"

// IdGenerator generates monotonically increasing uint32 IDs in a concurrency-safe
// manner. Each call to Id returns the next ID. The starting value is set at
// construction and the first Id() returns startValue+1.
type IdGenerator struct {
	start uint32
	id    atomic.Uint32
}

// NewIdGenerator creates an IdGenerator that will generate IDs starting from
// startValue+1. The generator is safe for concurrent use.
//
// Parameters:
//   - startValue: The value to initialize the counter to; the first Id() will
//     return startValue+1
//
// Returns:
//   - A new IdGenerator instance
func NewIdGenerator(startValue uint32) *IdGenerator {
	gen := &IdGenerator{
		start: startValue,
	}
	gen.id.Store(startValue)
	return gen
}

// Id returns the next unique ID by atomically incrementing the internal counter.
// It is safe for concurrent use by multiple goroutines.
//
// Returns:
//   - The next uint32 ID
func (l *IdGenerator) Id() uint32 {
	return l.id.Add(1)
}
