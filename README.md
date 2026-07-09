# goconc

[![CI](https://github.com/SimiyuWafulah/goconc/actions/workflows/ci.yml/badge.svg)](https://github.com/SimiyuWafulah/goconc/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SimiyuWafulah/goconc.svg)](https://pkg.go.dev/github.com/SimiyuWafulah/goconc)
[![Go Report Card](https://goreportcard.com/badge/github.com/SimiyuWafulah/goconc)](https://goreportcard.com/report/github.com/SimiyuWafulah/goconc)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Small, focused concurrency-safety primitives for Go. Each package exists
to fix one specific, real footgun — races, deadlocks, or goroutine leaks
— rather than to be a general-purpose framework.

Import only what you need; the packages don't depend on each other.

```bash
go get github.com/SimiyuWafulah/goconc
```

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
| [`pool`](./pool) | `github.com/SimiyuWafulah/goconc/pool` | Bounded, context-aware worker pool — caps concurrency, cancellation actually stops queued work |
| [`once`](./once) | `github.com/SimiyuWafulah/goconc/once` | `Group`: safe run-and-collect-first-error helper, replacing hand-rolled `WaitGroup` + shared error variable |
| [`deadline`](./deadline) | `github.com/SimiyuWafulah/goconc/deadline` | A mutex with an acquisition timeout — turns silent deadlocks into a loggable error |
| [`leakcheck`](./leakcheck) | `github.com/SimiyuWafulah/goconc/leakcheck` | Test helper that fails a test if it leaves goroutines running |

Each package has its own doc comment (readable via `go doc` or
[pkg.go.dev](https://pkg.go.dev/github.com/SimiyuWafulah/goconc)) and a
runnable example under [`examples/`](./examples).

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
an error, the pool's context is canceled, so well-behaved jobs still in
flight can bail out instead of running to completion pointlessly.
`Wait()` blocks until every worker has actually exited — a passing
`Wait()` is a guarantee that nothing from this pool is still running.

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
