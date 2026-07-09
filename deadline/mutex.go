// Package deadline provides a mutex that fails loudly instead of hanging
// forever when a lock can't be acquired within a timeout.
//
// A plain sync.Mutex gives no signal when something has gone wrong --
// a genuine deadlock and a slow-but-fine critical section look identical
// from the outside: everything just hangs. Mutex turns "hangs forever"
// into "returns ErrTimeout after N seconds", which is far easier to
// notice, log, and alert on in production.
package deadline

import (
	"errors"
	"time"
)

// ErrTimeout is returned by Lock when the timeout elapses before the lock
// could be acquired.
var ErrTimeout = errors.New("deadline: timed out waiting for lock")

// Mutex is a mutual-exclusion lock with a fixed acquisition timeout.
// The zero value is not usable; construct one with NewMutex.
type Mutex struct {
	timeout time.Duration
	ch      chan struct{} // buffered with capacity 1; acts as a 1-slot semaphore
}

// NewMutex creates a Mutex that gives up on acquisition after timeout.
// A timeout <= 0 means "wait forever", matching sync.Mutex's behavior --
// useful if you want the safety net disabled temporarily without changing
// call sites.
func NewMutex(timeout time.Duration) *Mutex {
	ch := make(chan struct{}, 1)
	ch <- struct{}{}
	return &Mutex{timeout: timeout, ch: ch}
}

// Lock attempts to acquire the lock, returning ErrTimeout if it could not
// be acquired within the configured timeout.
func (m *Mutex) Lock() error {
	if m.timeout <= 0 {
		<-m.ch
		return nil
	}

	timer := time.NewTimer(m.timeout)
	defer timer.Stop()

	select {
	case <-m.ch:
		return nil
	case <-timer.C:
		return ErrTimeout
	}
}

// Unlock releases the lock. Calling Unlock on a Mutex that isn't
// currently locked (by this goroutine or otherwise) will panic, matching
// sync.Mutex's semantics.
func (m *Mutex) Unlock() {
	select {
	case m.ch <- struct{}{}:
	default:
		panic("deadline: Unlock of unlocked Mutex")
	}
}
