package deadline

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestLockUnlockBasic(t *testing.T) {
	m := NewMutex(time.Second)
	if err := m.Lock(); err != nil {
		t.Fatalf("Lock() = %v; want nil", err)
	}
	m.Unlock()
}

func TestLockTimesOutWhenHeld(t *testing.T) {
	m := NewMutex(50 * time.Millisecond)
	if err := m.Lock(); err != nil {
		t.Fatalf("first Lock() = %v; want nil", err)
	}
	// Do not unlock -- simulate a stuck critical section.

	start := time.Now()
	err := m.Lock()
	elapsed := time.Since(start)

	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("second Lock() = %v; want ErrTimeout", err)
	}
	if elapsed < 50*time.Millisecond {
		t.Fatalf("returned too early: %v", elapsed)
	}
}

func TestUnlockOfUnlockedPanics(t *testing.T) {
	m := NewMutex(time.Second)
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on Unlock of unlocked Mutex")
		}
	}()
	m.Unlock()
}

func TestZeroTimeoutWaitsForever(t *testing.T) {
	m := NewMutex(0)
	if err := m.Lock(); err != nil {
		t.Fatalf("Lock() = %v; want nil", err)
	}

	unlocked := make(chan struct{})
	go func() {
		time.Sleep(30 * time.Millisecond)
		m.Unlock()
		close(unlocked)
	}()

	if err := m.Lock(); err != nil {
		t.Fatalf("second Lock() = %v; want nil (should wait, not time out)", err)
	}
	<-unlocked
}

// TestConcurrentLockUnlock is meant to be run with -race.
func TestConcurrentLockUnlock(t *testing.T) {
	m := NewMutex(2 * time.Second)
	var counter int
	var wg sync.WaitGroup

	const goroutines = 20
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if err := m.Lock(); err != nil {
					t.Errorf("Lock() = %v; want nil", err)
					return
				}
				counter++
				m.Unlock()
			}
		}()
	}
	wg.Wait()

	if counter != goroutines*50 {
		t.Fatalf("counter = %d; want %d", counter, goroutines*50)
	}
}
