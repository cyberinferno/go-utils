// Package safemap provides a type-safe, concurrent map built on sync.Map.
// SafeMap supports arbitrary comparable keys and any value type, with a
// consistent API for storing, loading, deleting, and iterating entries.
package safemap

import "sync"

// SafeMap is a concurrent map that is safe for use by multiple goroutines.
// It wraps sync.Map and exposes a generic, type-safe API. Keys must be
// comparable (as defined by the comparable constraint); values may be any type.
//
// SafeMap must not be copied after first use. Store and Load operations
// are amortized O(1). Len and Range are O(n) in the number of entries.
type SafeMap[K comparable, V any] struct {
	m sync.Map
}

// Store sets the value for key k. It overwrites any existing value for k.
//
// Parameters:
//   - k: The key to store
//   - v: The value to associate with k
func (m *SafeMap[K, V]) Store(k K, v V) {
	m.m.Store(k, v)
}

// Set sets the value for key k. It is equivalent to Store and overwrites
// any existing value for k.
//
// Parameters:
//   - k: The key to set
//   - v: The value to associate with k
func (m *SafeMap[K, V]) Set(k K, v V) {
	m.Store(k, v)
}

// Load returns the value for key k and a boolean indicating whether the key
// was present. If the key is not in the map, the value is the zero value
// for V and the boolean is false.
//
// Parameters:
//   - k: The key to look up
//
// Returns:
//   - The value associated with k, or the zero value of V if not found
//   - true if the key was present, false otherwise
func (m *SafeMap[K, V]) Load(k K) (V, bool) {
	v, found := m.m.Load(k)
	if !found {
		var empty V
		return empty, found
	}

	return v.(V), found
}

// Get returns the value for key k. It is equivalent to Load.
//
// Parameters:
//   - k: The key to look up
//
// Returns:
//   - The value associated with k, or the zero value of V if not found
//   - true if the key was present, false otherwise
func (m *SafeMap[K, V]) Get(k K) (V, bool) {
	return m.Load(k)
}

// Delete removes the entry for key k. It is safe to call for a key that
// is not in the map; the call is a no-op in that case.
//
// Parameters:
//   - k: The key to delete
func (m *SafeMap[K, V]) Delete(k K) {
	m.m.Delete(k)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, Range stops the iteration. Range does not support
// concurrent modification of the map from within f; the behavior is
// undefined if the map is modified during iteration.
//
// Parameters:
//   - f: Function called for each entry; return false to stop iteration
func (m *SafeMap[K, V]) Range(f func(k K, v V) bool) {
	m.m.Range(func(k, v interface{}) bool {
		return f(k.(K), v.(V))
	})
}

// Len returns the number of entries in the map. It iterates over all entries
// to compute the count; use sparingly on large maps.
//
// Returns:
//   - The number of key-value pairs in the map
func (m *SafeMap[K, V]) Len() int {
	length := 0
	m.Range(func(k K, v V) bool {
		length++
		return true
	})

	return length
}

// Has reports whether key k is present in the map.
//
// Parameters:
//   - k: The key to check
//
// Returns:
//   - true if the key is in the map, false otherwise
func (m *SafeMap[K, V]) Has(k K) bool {
	_, found := m.Load(k)
	return found
}

// NewSafeMap returns a new SafeMap ready for use. The map is empty and
// safe for concurrent use by multiple goroutines.
//
// Returns:
//   - A pointer to a new SafeMap[K, V]
func NewSafeMap[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{}
}
