package leakcheck

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCheckPassesWhenClean(t *testing.T) {
	done := Check(t)
	defer done()

	ch := make(chan struct{})
	go func() {
		<-ch // wait to be told to stop
	}()
	close(ch) // let it exit
	time.Sleep(20 * time.Millisecond)
}

// fakeTB is a minimal testing.TB that records failures instead of
// stopping the test, so we can assert on leakcheck's own behavior
// without actually failing this test suite when we intentionally leak.
type fakeTB struct {
	testing.TB
	mu     sync.Mutex
	errors []string
}

func (f *fakeTB) Helper() {}

func (f *fakeTB) Errorf(format string, args ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errors = append(f.errors, format)
}

func (f *fakeTB) hasErrors() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.errors) > 0
}

func TestCheckDetectsLeak(t *testing.T) {
	// Speed up the poll loop for this test only.
	oldTimeout, oldInterval := timeout, pollInterval
	timeout = 200 * time.Millisecond
	pollInterval = 5 * time.Millisecond
	defer func() { timeout, pollInterval = oldTimeout, oldInterval }()

	ft := &fakeTB{}
	done := Check(ft)

	block := make(chan struct{})
	go func() {
		<-block // deliberately never closed within the test
	}()
	defer close(block) // clean up after the assertion so we don't leak for real

	done()

	if !ft.hasErrors() {
		t.Fatalf("expected Check to report a leaked goroutine, got no errors")
	}
	if !strings.Contains(ft.errors[0], "goroutine count grew") {
		t.Fatalf("unexpected error message: %q", ft.errors[0])
	}
}
