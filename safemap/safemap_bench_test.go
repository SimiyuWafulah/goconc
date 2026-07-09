package safemap

import (
	"strconv"
	"sync"
	"testing"
)

// manualMap is the hand-rolled RWMutex+map pattern safemap.Map replaces,
// used here purely as a benchmark baseline.
type manualMap struct {
	mu sync.RWMutex
	m  map[string]int
}

func newManualMap() *manualMap {
	return &manualMap{m: make(map[string]int)}
}

func (mm *manualMap) Set(k string, v int) {
	mm.mu.Lock()
	mm.m[k] = v
	mm.mu.Unlock()
}

func (mm *manualMap) Get(k string) (int, bool) {
	mm.mu.RLock()
	v, ok := mm.m[k]
	mm.mu.RUnlock()
	return v, ok
}

func BenchmarkManualMap_ReadHeavy(b *testing.B) {
	mm := newManualMap()
	for i := 0; i < 1000; i++ {
		mm.Set(strconv.Itoa(i), i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			mm.Get(strconv.Itoa(i % 1000))
			i++
		}
	})
}

func BenchmarkSafeMap_ReadHeavy(b *testing.B) {
	sm := New[string, int]()
	for i := 0; i < 1000; i++ {
		sm.Set(strconv.Itoa(i), i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sm.Get(strconv.Itoa(i % 1000))
			i++
		}
	})
}

func BenchmarkManualMap_WriteHeavy(b *testing.B) {
	mm := newManualMap()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			mm.Set(strconv.Itoa(i%1000), i)
			i++
		}
	})
}

func BenchmarkSafeMap_WriteHeavy(b *testing.B) {
	sm := New[string, int]()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sm.Set(strconv.Itoa(i%1000), i)
			i++
		}
	})
}

// BenchmarkSafeMap_Range measures the cost of Range's snapshot-then-iterate
// approach, since that's the one place safemap.Map does more work than a
// naive "hold the lock for the whole loop" implementation would.
func BenchmarkSafeMap_Range(b *testing.B) {
	sm := New[string, int]()
	for i := 0; i < 1000; i++ {
		sm.Set(strconv.Itoa(i), i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm.Range(func(k string, v int) bool { return true })
	}
}