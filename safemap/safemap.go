// Package safemap provides a generic, goroutine-safe map.
//
// The standard pattern of `map[K]V` guarded by a `sync.RWMutex` is easy to
// get wrong in small ways: forgetting to RLock a read, holding the lock
// across a callback and deadlocking, or racing during iteration. Map[K, V]
// wraps that pattern once, correctly, behind a small API.
package safemap

import "sync"

// Map is a goroutine-safe map from keys of type K to values of type V.
// The zero value is not usable; construct one with New.
type Map[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

// New creates an empty, ready-to-use Map.
func New[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{m: make(map[K]V)}
}

// Set stores value under key, overwriting any existing value.
func (s *Map[K, V]) Set(key K, value V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
}

// Get returns the value stored under key, and whether it was present.
func (s *Map[K, V]) Get(key K) (V, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[key]
	return v, ok
}

// Delete removes key from the map. It is a no-op if key is not present.
func (s *Map[K, V]) Delete(key K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
}

// Len returns the number of entries currently stored.
func (s *Map[K, V]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}

// Range calls fn for each key/value pair in the map. If fn returns false,
// Range stops early.
//
// Range takes a snapshot of the map before iterating (rather than holding
// the lock for the duration), so fn is free to call back into the same Map
// -- including Set or Delete -- without deadlocking. The tradeoff is that
// fn may observe a slightly stale view under heavy concurrent writes.
func (s *Map[K, V]) Range(fn func(key K, value V) bool) {
	s.mu.RLock()
	snapshot := make(map[K]V, len(s.m))
	for k, v := range s.m {
		snapshot[k] = v
	}
	s.mu.RUnlock()

	for k, v := range snapshot {
		if !fn(k, v) {
			return
		}
	}
}

// Keys returns a snapshot slice of all keys currently in the map.
func (s *Map[K, V]) Keys() []K {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]K, 0, len(s.m))
	for k := range s.m {
		keys = append(keys, k)
	}
	return keys
}
