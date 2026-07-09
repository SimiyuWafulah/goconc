package once

import (
	"errors"
	"sync/atomic"
	"testing"
)

func TestGroupAllSucceed(t *testing.T) {
	var g Group
	var count int64

	for i := 0; i < 20; i++ {
		g.Go(func() error {
			atomic.AddInt64(&count, 1)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		t.Fatalf("Wait() = %v; want nil", err)
	}
	if got := atomic.LoadInt64(&count); got != 20 {
		t.Fatalf("count = %d; want 20", got)
	}
}

func TestGroupCollectsFirstError(t *testing.T) {
	var g Group
	wantErr := errors.New("boom")

	g.Go(func() error { return nil })
	g.Go(func() error { return wantErr })
	g.Go(func() error { return nil })

	if err := g.Wait(); !errors.Is(err, wantErr) {
		t.Fatalf("Wait() = %v; want %v", err, wantErr)
	}
}

func TestGroupZeroValueUsable(t *testing.T) {
	var g Group // no constructor call
	g.Go(func() error { return nil })
	if err := g.Wait(); err != nil {
		t.Fatalf("Wait() = %v; want nil", err)
	}
}

// TestGroupConcurrentErrors is meant to be run with -race: many goroutines
// return errors "simultaneously" and Wait() must not race on the stored
// error value.
func TestGroupConcurrentErrors(t *testing.T) {
	var g Group
	for i := 0; i < 50; i++ {
		g.Go(func() error {
			return errors.New("one of many")
		})
	}
	if err := g.Wait(); err == nil {
		t.Fatalf("Wait() = nil; want an error")
	}
}
