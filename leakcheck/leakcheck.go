// Package leakcheck provides a test helper that fails a test if it leaves
// goroutines running after it finishes.
//
// A test can pass -- and even pass with -race -- while quietly leaking a
// goroutine that outlives it (e.g. a worker blocked forever on an
// unbuffered channel nobody reads from again). Those leaks don't show up
// until they accumulate in a long-running process. Check(t) catches them
// at the point they're introduced.
package leakcheck

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

// pollInterval and timeout control how long Check waits for goroutine
// counts to settle before declaring a leak. Goroutines started by things
// like the runtime or GC can take a moment to exit even in non-leaking
// code, so Check polls rather than checking exactly once.
//
// These are package-level variables (not constants) so this package's own
// tests can shrink them for speed; library users should not need to
// touch them.
var (
	pollInterval = 10 * time.Millisecond
	timeout      = 2 * time.Second
)

// Check records the current goroutine count and returns a function that
// should be deferred immediately: `defer leakcheck.Check(t)()`.
//
// When the returned function runs, it polls runtime.NumGoroutine() until
// it returns to (or below) the recorded baseline, or until a timeout
// elapses -- at which point it fails t with a goroutine dump to help
// identify the leak.
func Check(t testing.TB) func() {
	t.Helper()
	before := runtime.NumGoroutine()

	return func() {
		t.Helper()

		deadline := time.Now().Add(timeout)
		for {
			after := runtime.NumGoroutine()
			if after <= before {
				return
			}
			if time.Now().After(deadline) {
				t.Errorf(
					"leakcheck: goroutine count grew from %d to %d and did not settle within %s\n%s",
					before, after, timeout, dumpGoroutines(),
				)
				return
			}
			time.Sleep(pollInterval)
		}
	}
}

// dumpGoroutines returns a formatted stack dump of all running
// goroutines, to help pinpoint what's still running.
func dumpGoroutines() string {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	return "goroutine dump:\n" + strings.TrimSpace(string(buf[:n])) + fmt.Sprintf("\n(%d bytes)", n)
}
