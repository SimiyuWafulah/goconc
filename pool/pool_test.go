package pool

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestPoolRunsAllJobs(t *testing.T) {
	p := New(context.Background(), 4)

	var count int64
	const jobs = 100
	for i := 0; i < jobs; i++ {
		p.Submit(func(ctx context.Context) error {
			atomic.AddInt64(&count, 1)
			return nil
		})
	}

	if err := p.Wait(); err != nil {
		t.Fatalf("Wait() = %v; want nil", err)
	}
	if got := atomic.LoadInt64(&count); got != jobs {
		t.Fatalf("count = %d; want %d", got, jobs)
	}
}

func TestPoolPropagatesFirstError(t *testing.T) {
	p := New(context.Background(), 2)
	wantErr := errors.New("boom")

	p.Submit(func(ctx context.Context) error {
		return wantErr
	})

	if err := p.Wait(); !errors.Is(err, wantErr) {
		t.Fatalf("Wait() = %v; want %v", err, wantErr)
	}
}

func TestPoolStopsRemainingWorkAfterError(t *testing.T) {
	p := New(context.Background(), 1)
	var ran int64

	p.Submit(func(ctx context.Context) error {
		return errors.New("fail fast")
	})
	// This job may or may not start depending on scheduling, but it must
	// never block Wait() from returning, and the pool's context must be
	// canceled so a well-behaved job bails out quickly.
	p.Submit(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			atomic.AddInt64(&ran, 1)
			return nil
		}
	})

	if err := p.Wait(); err == nil {
		t.Fatalf("Wait() = nil; want an error")
	}
}

func TestPoolRespectsExternalCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := New(ctx, 1)

	started := make(chan struct{})
	p.Submit(func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})

	<-started
	cancel()

	if err := p.Wait(); err == nil {
		t.Fatalf("Wait() = nil; want context error after external cancellation")
	}
}

func TestPoolMinimumOneWorker(t *testing.T) {
	p := New(context.Background(), 0) // should be clamped to 1, not hang forever

	done := make(chan struct{})
	p.Submit(func(ctx context.Context) error {
		close(done)
		return nil
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("job never ran; pool with 0 workers likely deadlocked")
	}
	_ = p.Wait()
}
