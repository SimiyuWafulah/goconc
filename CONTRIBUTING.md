# Contributing to goconc

Thanks for considering a contribution. This project stays small and
focused on purpose — here's how to work with it.

## Philosophy

Every package in this repo exists because of a real, observed bug or
footgun (races, deadlocks, goroutine leaks), not a hypothetical one.
Before adding a new package, open a
[feature request](.github/ISSUE_TEMPLATE/feature_request.md) describing
the concrete problem it solves. PRs that add speculative API surface
without a motivating bug/pattern are likely to get pushback.

## Local setup

```bash
git clone https://github.com/SimiyuWafulah/goconc.git
cd goconc
go build ./...
go test -race ./...
```

No external dependencies, no `go generate` step, no Docker required —
if `go build` and `go test` both work, your environment is fully set up.

Opening this repo in a GitHub Codespace also works out of the box; the
Go toolchain is preinstalled in the default Codespaces image.

## Before opening a PR

Run, in order:

```bash
gofmt -l .              # should print nothing
go vet ./...             # should print nothing
go build ./...
go test -race -count=1 ./...
```

If you touched a package that has a runnable example under
`examples/<pkg>`, run it manually and confirm the output still matches
what the example's comments claim:

```bash
go run ./examples/<pkg>
```

This project favors this kind of direct, manual verification over
relying solely on the automated test suite — CI runs `go test -race`
as its one hard gate, but "did you actually run it and look at the
output" catches things tests don't.

## Style

- Every exported type/func needs a doc comment. Package-level doc
  comments (at the top of the primary file) should explain *why* the
  package exists, not just what it does.
- If a change touches shared state across goroutines, it needs a test
  that exercises concurrent access and is run under `-race` in CI —
  see `safemap`'s `TestConcurrentAccess` or `pool`'s tests for the
  pattern.
- Keep the API surface small. If you're unsure whether something
  belongs in the public API vs. staying internal, default to internal.

## Commit style

Commits should be scoped and descriptive:
`feat(pool): add Submit timeout option`, `fix(safemap): Range snapshot
copies value type correctly`, `docs: clarify deadline.Mutex zero-timeout
behavior`. Squash-merge is fine for PRs with many small WIP commits, but
keep the final commit message meaningful.

## Reporting bugs

Please use the
[bug report template](.github/ISSUE_TEMPLATE/bug_report.md) and include
a minimal reproduction — ideally something under 30 lines that can be
pasted into a `go run` file. If it's a concurrency bug, let us know
whether you saw it with `-race` or without.
