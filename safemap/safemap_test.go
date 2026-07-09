package safemap

import (
	"sync"
	"testing"
)

func TestSetGet(t *testing.T) {
	m := New[string, int]()
	m.Set("a", 1)

	v, ok := m.Get("a")
	if !ok || v != 1 {
		t.Fatalf("Get(a) = %d, %v; want 1, true", v, ok)
	}

	_, ok = m.Get("missing")
	if ok {
		t.Fatalf("Get(missing) ok = true; want false")
	}
}

func TestDelete(t *testing.T) {
	m := New[string, int]()
	m.Set("a", 1)
	m.Delete("a")

	if _, ok := m.Get("a"); ok {
		t.Fatalf("expected key to be deleted")
	}

	// Delete on a missing key must not panic.
	m.Delete("never-existed")
}

func TestLen(t *testing.T) {
	m := New[string, int]()
	if got := m.Len(); got != 0 {
		t.Fatalf("Len() = %d; want 0", got)
	}
	m.Set("a", 1)
	m.Set("b", 2)
	if got := m.Len(); got != 2 {
		t.Fatalf("Len() = %d; want 2", got)
	}
}

func TestRangeCanMutateSafely(t *testing.T) {
	m := New[int, int]()
	for i := 0; i < 10; i++ {
		m.Set(i, i*i)
	}

	// Regression test: calling Set/Delete from inside the Range callback
	// must not deadlock, since Range snapshots before iterating.
	m.Range(func(k, v int) bool {
		m.Set(k, v+1)
		return true
	})

	if got := m.Len(); got != 10 {
		t.Fatalf("Len() after Range = %d; want 10", got)
	}
}

func TestRangeEarlyStop(t *testing.T) {
	m := New[int, int]()
	for i := 0; i < 5; i++ {
		m.Set(i, i)
	}

	visited := 0
	m.Range(func(k, v int) bool {
		visited++
		return false // stop after first
	})

	if visited != 1 {
		t.Fatalf("visited = %d; want 1", visited)
	}
}

// TestConcurrentAccess is meant to be run with -race. It hammers the map
// from many goroutines doing a mix of writes, reads, deletes, and range
// calls concurrently.
func TestConcurrentAccess(t *testing.T) {
	m := New[int, int]()
	const goroutines = 50
	const opsPerGoroutine = 200

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				key := (id + i) % 20
				switch i % 4 {
				case 0:
					m.Set(key, i)
				case 1:
					m.Get(key)
				case 2:
					m.Delete(key)
				case 3:
					m.Range(func(k, v int) bool { return true })
				}
			}
		}(g)
	}

	wg.Wait()
}
