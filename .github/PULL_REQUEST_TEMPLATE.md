## What does this change?

<!-- One or two sentences: what changed and why. -->

## Verification

- [ ] `go build ./...`
- [ ] `go vet ./...`
- [ ] `go test -race ./...`
- [ ] `gofmt -l .` is clean
- [ ] `golangci-lint run ./...` is clean
- [ ] `govulncheck ./...` is clean (see CONTRIBUTING.md for the versioned command)
- [ ] Added/updated tests for the behavior change

## Docs

- [ ] If this changes user-facing behavior, I updated `docs/en/` (the source of truth)
- [ ] If `docs/en/` changed, I noted translation debt for `docs/ru/` and `docs/zh/` (a one-line `## Translation debt` note below is enough — translations are tracked as follow-up, not a merge blocker)
- [ ] If this changes the rendered site (templates, CSS, generator code), I ran `make site-serve` and verified the output

## Notes for the reviewer

<!-- Anything that needs context: API quirks discovered, live-verification
     cassettes added/changed, breaking changes, follow-up work left for later. -->
