## What this changes

<!-- One or two sentences: what does this PR do, and why. -->

## Checklist

- [ ] `go build ./...` passes
- [ ] `go test -race ./...` passes
- [ ] `go vet ./...` passes with no warnings
- [ ] `gofmt -l .` produces no output (code is formatted)
- [ ] New/changed exported identifiers have doc comments
- [ ] If this touches a package with an `examples/<pkg>` demo, the demo
      was run manually and its output checked
- [ ] Tests were added for new behavior, including a concurrency stress
      test if the change touches shared state

## Related issue

<!-- Link the issue this addresses, if any. -->
