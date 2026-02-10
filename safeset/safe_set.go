package safeset

import "sync"

// SafeSet is a thread-safe set that stores a collection of unique elements of
// comparable type T. It is safe for concurrent use by multiple goroutines.
type SafeSet[T comparable] struct {
	m map[T]struct{}
	sync.RWMutex
}

// NewSafeSet creates and returns a new empty SafeSet.
func NewSafeSet[T comparable]() *SafeSet[T] {
	return &SafeSet[T]{m: make(map[T]struct{})}
}

// Add adds an element to the set.
//
// Parameters:
//   - value: The element to add
func (s *SafeSet[T]) Add(value T) {
	s.Lock()
	defer s.Unlock()
	s.m[value] = struct{}{}
}

// Remove removes an element from the set.
//
// Parameters:
//   - value: The element to remove
func (s *SafeSet[T]) Remove(value T) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, value)
}

// Contains reports whether the set contains the given element.
//
// Parameters:
//   - value: The element to look up
//
// Returns:
//   - true if the set contains value, false otherwise
func (s *SafeSet[T]) Contains(value T) bool {
	s.RLock()
	defer s.RUnlock()
	_, ok := s.m[value]
	return ok
}

// Size returns the number of elements in the set.
//
// Returns:
//   - The number of elements in the set
func (s *SafeSet[T]) Size() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.m)
}

// Intersection returns a new set containing only the elements that are present
// in both this set and the other set.
//
// Parameters:
//   - other: The other set to intersect with
//
// Returns:
//   - A new SafeSet containing the intersection of the two sets
func (s *SafeSet[T]) Intersection(other *SafeSet[T]) *SafeSet[T] {
	result := NewSafeSet[T]()
	for k := range s.m {
		if _, ok := other.m[k]; ok {
			result.Add(k)
		}
	}
	return result
}

// Union returns a new set containing all elements that are in this set, the
// other set, or both.
//
// Parameters:
//   - other: The other set to union with
//
// Returns:
//   - A new SafeSet containing the union of the two sets
func (s *SafeSet[T]) Union(other *SafeSet[T]) *SafeSet[T] {
	s.RLock()
	defer s.RUnlock()
	other.RLock()
	defer other.RUnlock()
	result := NewSafeSet[T]()
	for k := range s.m {
		result.Add(k)
	}
	for k := range other.m {
		if _, ok := s.m[k]; !ok {
			result.Add(k)
		}
	}
	return result
}

// Reset removes all elements from the set, leaving it empty.
func (s *SafeSet[T]) Reset() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[T]struct{})
}

// Range calls the function f for each element in the set. Iteration stops if f
// returns false. The behavior is undefined if f modifies the set.
//
// Parameters:
//   - f: Function called for each element; return false to stop iteration
func (s *SafeSet[T]) Range(f func(value T) bool) {
	s.RLock()
	defer s.RUnlock()
	for k := range s.m {
		if !f(k) {
			break
		}
	}
}
