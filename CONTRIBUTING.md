# Contributing

Thanks for considering a contribution. This is a Go CLI + client library for
the Z.AI (Zhipu AI) API. Start with [docs/architecture.md](docs/architecture.md)
if you want the package layout and design rationale before diving in, and
[docs/roadmap.md](docs/roadmap.md) for known gaps that are good first
contributions. A few things that make this repo different from a typical Go
project:

## The live-verification convention

Z.AI's official docs (docs.z.ai / docs.bigmodel.cn) and SDKs sometimes
disagree with each other, or with what the live API actually returns. Rather
than trust documentation alone, this project verifies request/response shapes
against real API calls and records the interaction as a
[go-vcr](https://github.com/dnaeon/go-vcr) cassette under
`pkg/client/testdata/cassettes/`, replayed in `ModeReplayOnly` so tests never
hit the network.

If you're adding a new endpoint or changing a request/response type:

- Prefer a cassette-backed test over a hand-written fixture when you can
  record one — it proves the types parse what the server actually sends.
- Name cassettes after what they call (`agents_invoke.yaml`), not when/why
  they were recorded.
- **Redact credentials before committing a cassette.** Never commit a real
  `Authorization` header, API key, or other account-identifying data. Check
  `grep -n "Bearer " your_cassette.yaml` and confirm it reads
  `Bearer REDACTED` before opening a PR.
- If you can't record a live cassette (no account, insufficient balance,
  etc.), say so in the PR — a `NOT VERIFIED LIVE` doc comment on the type is
  fine and expected; see existing examples in `pkg/client/agents.go`.

## Before opening a PR

```sh
go build ./...
go vet ./...
gofmt -l .          # must be empty
go test -race ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

All of the above run in CI; a green PR is a merged PR.

## Code conventions

- Every service method takes `context.Context` and propagates it through to
  the HTTP call — no exceptions for "it's just a quick helper."
- Required fields are validated before a request is built, not left to the
  API's error response.
- Don't add a new `http.Client` anywhere — every request goes through
  `Client.doRequest`/`doRequestBase`/`sendMultipart` so retry, timeout, and
  error parsing stay centralized.
- Keep exported contracts stable; if a change is breaking, call it out
  explicitly in the PR description.

## Reporting a security issue

See [SECURITY.md](SECURITY.md) — please don't open a public issue for a
vulnerability.
