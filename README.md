# goconc

[![CI](https://github.com/SimiyuWafulah/goconc/actions/workflows/ci.yml/badge.svg)](https://github.com/SimiyuWafulah/goconc/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SimiyuWafulah/goconc.svg)](https://pkg.go.dev/github.com/SimiyuWafulah/goconc)
[![Go Report Card](https://goreportcard.com/badge/github.com/SimiyuWafulah/goconc)](https://goreportcard.com/report/github.com/SimiyuWafulah/goconc)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Small, focused concurrency-safety primitives for Go. Each package exists
to fix one specific, real footgun — races, deadlocks, or goroutine leaks
— rather than to be a general-purpose framework.

Import only what you need; the packages don't depend on each other.

> **Status: v0.x.** The API is usable now but may still change before a
> `v1.0.0` is tagged. Breaking changes will be called out in release
> notes; feedback and issues are welcome.

## Why this exists

Go makes concurrency easy to *write* and easy to get subtly *wrong*.
The common failure modes are well known but keep recurring:

- a `map` guarded by a mutex where someone forgets `RLock` on a read path
- `go func(){}()` fired in a loop with no cap, no cancellation, and no
  way to know when (or if) they've all finished
- a shared `var firstErr error` written from multiple goroutines with no
  lock around it
- a lock that's held too long or never released, and the program just
  hangs with no signal about why
- a test that passes green while quietly leaving a goroutine running

Every package below was written to make one of those mistakes hard to
make by construction. They came out of real bugs found while reviewing
`go-caching-proxy` (a race condition on the cache store and a deadlock
under concurrent writes) — this isn't a hypothetical exercise.

## Packages

| Package | Import path | Problem it solves |
|---|---|---|
| [`safemap`](./safemap) | `github.com/SimiyuWafulah/goconc/safemap` | Generic, goroutine-safe map — no more hand-rolled `RWMutex` + `map` |
| [`pool`](./pool) | `github.com/SimiyuWafulah/goconc/pool` | Bounded, context-aware worker pool — caps concurrency; queued work is canceled before it starts, and running work can stop early if it respects the provided context |
| [`once`](./once) | `github.com/SimiyuWafulah/goconc/once` | `Group`: safe run-and-collect-first-error helper, replacing hand-rolled `WaitGroup` + shared error variable |
| [`deadline`](./deadline) | `github.com/SimiyuWafulah/goconc/deadline` | A mutex with an acquisition timeout — turns silent deadlocks into a loggable error |
| [`leakcheck`](./leakcheck) | `github.com/SimiyuWafulah/goconc/leakcheck` | Test helper that fails a test if it leaves goroutines running |

Each package has its own doc comment (readable via `go doc` or
[pkg.go.dev](https://pkg.go.dev/github.com/SimiyuWafulah/goconc)) and a
runnable example under [`examples/`](./examples).

### Why not just use the standard library?

You often should — these packages only earn their place where the
plain standard-library pattern has a well-known sharp edge:

| Need | Plain standard library | goconc |
|---|---|---|
| Map shared across goroutines | `map` + `sync.RWMutex`, hand-rolled | `safemap.Map[K, V]` |
| Bounded concurrent work with cancellation | `go func(){}()` in a loop, no cap, no cancellation | `pool.Pool` |
| Run several things, keep the first error | `sync.WaitGroup` + a shared error var (needs its own lock) | `once.Group` |
| Detect a lock that's stuck | not supported — `sync.Mutex.Lock()` blocks forever | `deadline.Mutex` |
| Catch a leaked goroutine in a test | not supported — tests just pass | `leakcheck.Check(t)` |

### When *not* to use this

- **`safemap`** — skip it if the map never crosses a goroutine boundary,
  or if you're on a single hot path where the tiny synchronization cost
  actually shows up in profiling. A plain `map` is faster when nothing
  else touches it concurrently.
- **`pool`** — skip it for a small, fixed, known-at-compile-time number
  of goroutines (e.g. spawning exactly 3 named workers) — plain
  goroutines plus a `sync.WaitGroup` are simpler and clearer there.
- **`deadline`** — Go deliberately omits timed mutexes; reach for this
  only when you specifically want "fail loud after N seconds" behavior
  (e.g. surfacing a stuck lock in production logs), not as a default
  replacement for `sync.Mutex`.
- **`leakcheck`** — it's a test-time tool; it adds a few seconds of
  polling per test in the worst case (when something actually leaks), so
  it's best added to tests that specifically exercise goroutine
  lifecycles, not blanket-applied everywhere.

---

### `safemap`

```go
m := safemap.New[string, int]()
m.Set("requests", 1)

v, ok := m.Get("requests")

m.Range(func(k string, v int) bool {
    fmt.Println(k, v)
    return true // return false to stop early
})
```

`Range` snapshots the map before iterating, so it's safe to call `Set`
or `Delete` on the same map from inside the callback — a naive
`RWMutex`-guarded map deadlocks if you try that while still holding the
read lock.

Run the example: `go run ./examples/safemap`

### `pool`

```go
p := pool.New(ctx, 5) // at most 5 jobs run at once

for _, item := range items {
    item := item
    p.Submit(func(ctx context.Context) error {
        return process(ctx, item)
    })
}

if err := p.Wait(); err != nil {
    // first error from any job; remaining queued work was canceled
}
```

Every job gets a `context.Context` tied to the pool. If any job returns
an error, the pool's context is canceled: queued work that hasn't
started yet is dropped, and running work can stop early if it checks
`ctx.Done()` (it's not preemptively killed — Go can't do that safely).
`Wait()` blocks until every worker has actually exited — a passing
`Wait()` is a guarantee that nothing from this pool is still running.

**Behavior worth knowing before you adopt it:**
- **Queue size:** `Submit` is unbuffered — it blocks until a worker
  picks up the job, the pool's context is canceled, or `Wait` has
  started closing the pool. There's no separate bounded queue to size;
  the worker count *is* the concurrency cap.
- **Backpressure:** because `Submit` blocks, a slow consumer naturally
  applies backpressure to whatever's calling `Submit` — it won't buffer
  unbounded work in memory.
- **Panics:** a panicking job crashes the worker goroutine like any
  other goroutine panic; `Pool` does not recover panics for you. Recover
  inside your own job function if you need that.
- **`Submit` after `Wait`:** undefined/unsupported — don't call `Submit`
  once you've called `Wait`. Treat the pool as single-phase: submit,
  then wait, don't interleave from multiple goroutines calling `Wait`.
- **`Submit` after cancellation:** returns immediately without running
  the job (it's dropped), since the pool is shutting down.

Run the example: `go run ./examples/pool`

### `once`

```go
var g once.Group

g.Go(func() error { return fetchProfile(ctx) })
g.Go(func() error { return fetchBilling(ctx) })
g.Go(func() error { return fetchPrefs(ctx) })

if err := g.Wait(); err != nil {
    // first error encountered, collected without a data race
}
```

The zero value is ready to use — no constructor needed.

Run the example: `go run ./examples/once`

### `deadline`

```go
m := deadline.NewMutex(2 * time.Second)

if err := m.Lock(); err != nil {
    // couldn't acquire within 2s -- log it, alert on it, whatever you'd
    // want to happen instead of hanging forever
}
defer m.Unlock()
```

Pass a timeout `<= 0` to fall back to blocking forever, matching
`sync.Mutex` — useful for disabling the safety net at a specific call
site without changing your code's shape.

Run the example: `go run ./examples/deadline`

### `leakcheck`

```go
func TestSomething(t *testing.T) {
    defer leakcheck.Check(t)()

    // ... test code that spawns goroutines ...
}
```

`Check` records the goroutine count when called; the deferred function
polls until the count settles back down (or fails the test with a
goroutine stack dump if it doesn't within ~2 seconds). Put it as the
*first* deferred call in a test so it runs *last*, after your other
cleanup.

## Installation & requirements

- Go 1.22 or newer (generics-based APIs; tested in CI on 1.22 and 1.23)
- No external dependencies

```bash
go get github.com/SimiyuWafulah/goconc
```

Then import whichever packages you need:

```go
import (
    "github.com/SimiyuWafulah/goconc/safemap"
    "github.com/SimiyuWafulah/goconc/pool"
)
```

## Testing this repo

```bash
go build ./...
go test -race -count=1 ./...
```

Every package that touches shared state has a concurrency stress test
meant to be run with `-race`; CI enforces this on every push and PR
(see [`.github/workflows/ci.yml`](.github/workflows/ci.yml)).

You can also run any package's example directly to see it behave in
practice rather than just reading the API docs:

```bash
go run ./examples/pool
```

## Contributing

Contributions are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md) for
local setup, the pre-PR checklist, and the project's philosophy
(packages should map to real observed bugs, not speculative API
surface). Use the
[bug report](.github/ISSUE_TEMPLATE/bug_report.md) or
[feature request](.github/ISSUE_TEMPLATE/feature_request.md) templates
when opening issues.

## License

[MIT](LICENSE)