// Package pool provides a bounded, context-aware worker pool.
//
// The common "go func() {...}()" pattern spawns an unbounded number of
// goroutines with no back-pressure and no guaranteed way to know when
// they're all done or to cancel them early. Pool fixes both: it caps
// concurrency to a fixed number of workers and ties every submitted job
// to a context. On cancellation, queued work that hasn't started yet is
// dropped, and in-flight work can stop early if it checks ctx.Done()
// (Go can't preemptively kill a running goroutine, so cooperative
// cancellation is still on the job function).
package pool

import (
	"context"
	"sync"
)

// Pool runs submitted jobs across a fixed number of worker goroutines.
type Pool struct {
	ctx    context.Context
	cancel context.CancelFunc
	jobs   chan func(context.Context) error

	wg      sync.WaitGroup
	errOnce sync.Once
	err     error

	closeOnce sync.Once
}

// New creates a Pool bound to ctx with the given number of workers.
// If ctx is canceled, queued and in-flight jobs are given the chance to
// observe cancellation via the context.Context passed to each job.
//
// workers must be >= 1; values < 1 are treated as 1.
func New(ctx context.Context, workers int) *Pool {
	if workers < 1 {
		workers = 1
	}

	ctx, cancel := context.WithCancel(ctx)
	p := &Pool{
		ctx:    ctx,
		cancel: cancel,
		jobs:   make(chan func(context.Context) error),
	}

	p.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go p.worker()
	}

	return p
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			if err := job(p.ctx); err != nil {
				p.errOnce.Do(func() {
					p.err = err
					p.cancel() // stop remaining work on first error
				})
			}
		}
	}
}

// Submit enqueues a job to be run by one of the pool's workers. Submit
// blocks until a worker is free, the pool's context is canceled, or the
// pool has been closed -- whichever happens first.
//
// Submit is safe to call from multiple goroutines. It is not safe to call
// Submit after Wait has been called.
func (p *Pool) Submit(job func(ctx context.Context) error) {
	select {
	case p.jobs <- job:
	case <-p.ctx.Done():
		// Pool is shutting down; drop the job rather than block forever.
	}
}

// Wait closes the pool to further submissions, waits for all in-flight and
// queued jobs to finish, and returns the first error returned by any job
// (or the context's error if the pool was canceled from outside), or nil
// if every job succeeded.
func (p *Pool) Wait() error {
	p.closeOnce.Do(func() {
		close(p.jobs)
	})
	p.wg.Wait()
	p.cancel() // release context resources

	if p.err != nil {
		return p.err
	}
	if err := p.ctx.Err(); err != nil && err != context.Canceled {
		return err
	}
	return nil
}
