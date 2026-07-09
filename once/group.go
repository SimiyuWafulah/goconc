// Package once provides a small, safe alternative to the common
// "sync.WaitGroup plus a shared error variable" pattern for running a
// batch of goroutines and collecting the first error.
//
// Writing that pattern by hand is an easy place to introduce a race: two
// goroutines writing to the same `var firstErr error` without a mutex, or
// forgetting wg.Add before the goroutine starts. Group makes the correct
// version the only version.
package once

import "sync"

// Group runs a set of functions concurrently and collects the first
// non-nil error returned by any of them. The zero value is ready to use.
type Group struct {
	wg      sync.WaitGroup
	mu      sync.Mutex
	err     error
	errOnce sync.Once
}

// Go runs fn in a new goroutine. Go must not be called after Wait has
// returned.
func (g *Group) Go(fn func() error) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if err := fn(); err != nil {
			g.errOnce.Do(func() {
				g.mu.Lock()
				g.err = err
				g.mu.Unlock()
			})
		}
	}()
}

// Wait blocks until every function passed to Go has returned, then
// returns the first non-nil error encountered (in the order errors
// occurred, not the order Go was called), or nil if none errored.
func (g *Group) Wait() error {
	g.wg.Wait()
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.err
}
