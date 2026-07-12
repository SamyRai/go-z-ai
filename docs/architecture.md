# Architecture

## Package layout

```
.                     CLI commands (package main, one file per command group:
                      chat.go, accounts_cli.go, coding_cli.go, ...)
pkg/client/           The Go library — one file per API service, no CLI/TUI dependency
pkg/accounts/         Multi-account credential store (~/.config/zai-client/accounts.json)
pkg/coding/           GLM Coding Plan credential store + per-tool config writers
pkg/usageview/        Pure presentation helpers (time windows, heat maps, formatting) —
                      shared by both the CLI and the TUI so their output can never drift
pkg/tui/              Bubble Tea terminal UI, one subpackage per tab
```

`pkg/client` has zero dependencies on anything CLI- or TUI-specific — it's
designed to be imported standalone (see the [Library Guide](library-guide.md)).
The CLI and TUI are both thin callers of it.

## The request facade

Every service method funnels through one of three `Client` methods —
services never build their own `http.Client` or issue a raw request:

```
doRequest(ctx, method, endpoint, body, result)
  → doRequestBase(ctx, baseURL, method, endpoint, body, result)   // for non-default base URLs
    → doRequestBaseKey(ctx, baseURL, apiKey, method, endpoint, body, result)  // for a different credential
```

This is what centralizes retry/backoff, error parsing, and auth for every
endpoint in one place. A service that needs a different base URL (Agents) or
a different credential (Embeddings/Moderations, which use `ChinaAPIKey`) calls
further down the chain — it never bypasses it.

`sendMultipart` is the parallel path for multipart/form-data uploads (files,
audio transcription, document parsing) — it does **not** retry, since
re-uploading a file on a transient failure is a caller decision, not a safe
default.

## Retry and timeout design

- `Config.Timeout` bounds dial + TLS handshake + waiting for response
  headers — deliberately **not** the whole `http.Client.Timeout`, which would
  truncate a long-running `CreateStream` SSE read partway through a
  generation.
- Retries apply to 429/5xx/network errors, with exponential backoff, jitter
  (up to 25%), and `Retry-After` header support, up to `Config.MaxRetries`
  (default 3). Every retry checks `ctx.Err()` first so a cancelled context
  aborts immediately instead of sleeping through a backoff first.
- Streaming (`CreateStream`) retries only the *connection* attempt — once the
  SSE stream has actually started, a mid-stream failure is surfaced to the
  caller, not silently retried (there's no way to know how much of the
  response the caller already consumed).

## Why some services hit a different host

| Service | Base URL | Why |
|---|---|---|
| Embeddings, Moderations | `open.bigmodel.cn` | The only platform that documents these endpoints — `api.z.ai`'s doc index doesn't mention either. Live-verified that a regular z.ai key authenticates identically on both platforms, so `Config.ChinaAPIKey` falls back to `Config.APIKey` by default. |
| Agents | `https://api.z.ai/api` (bare root, no `/paas/v4`) | Verified live — nesting `/v1/agents` under the chat-completions base 404s. |
| Everything else | `Config.BaseURL` (`/api/paas/v4`, or `/api/coding/paas/v4` for GLM Coding Plan accounts) | The general case. |

This project treats "documented" and "true" as separate claims and verifies
both before shipping a service — see the next section.

## The live-verification convention

Z.AI's own SDKs and docs sometimes disagree with each other, and sometimes
with what the live API actually returns (an endpoint documented as optional
that 400s without it; an error embedded in a 200 response body; a field typed
differently across two official SDKs). Rather than trust a single source,
new services here are checked against a real API call and the interaction is
recorded as a [go-vcr](https://github.com/dnaeon/go-vcr) cassette
(`pkg/client/testdata/cassettes/`), replayed in `ModeReplayOnly` so the test
suite never touches the network. `pkg/client/live_verification_test.go` is
the running log of what's been confirmed this way and why it mattered.

If you're extending a service, see
[Contributing § the live-verification convention](../CONTRIBUTING.md) before
you add a new cassette.

## Extending a structured lookup table

Several types in this codebase map a small, closed set of API-defined values
to human-readable metadata via a config table plus a lookup function, rather
than a chain of `if`/`switch` conditionals — `pkg/client/quota.go`'s
`quotaWindowConfigs`/`findWindowConfig` is the clearest example:

```go
var quotaWindowConfigs = []QuotaWindowConfig{
    {Type: QuotaTypeTokensLimit, UnitCode: UnitCodeHourly, Number: 5, Description: "5-hour rolling token window"},
    {Type: QuotaTypeTokensLimit, UnitCode: UnitCodeWeekly, Number: 1, Description: "weekly token window"},
    {Type: QuotaTypeTimeLimit, UnitCode: UnitCodeMonthly, Number: 1, Description: "monthly MCP tools quota"},
}
```

When Z.AI adds a new window type, the change is additive (append a row) and
localized, with a generic fallback description for anything not yet in the
table rather than a hard failure. The same pattern shows up for model
categorization (`pkg/client/models.go`'s `visionModelMarkers` — a single
source of truth so `isTextModel`/`isVisionModel` can never contradict each
other) and for account-type-to-endpoint resolution (`pkg/coding/plans.go`).
Prefer this shape over adding another conditional branch when you're adding a
new recognized value to an existing concept.

## Credential file safety

Both `pkg/accounts` (the multi-account store) and `pkg/coding` (the GLM
Coding Plan credential store, plus every third-party tool config it writes —
Claude Code, OpenCode, Crush, Factory Droid, Cursor) write via a temp-file
atomic write followed by rename, never a direct in-place write. A crash or
kill mid-write leaves the original file untouched instead of truncated —
important here specifically because several of these are *other programs'*
real config files being merged into, not files this project owns outright.
Files containing a key are created `0600`; directories `0700`.

## The TUI

`pkg/tui` is a [Bubble Tea](https://github.com/charmbracelet/bubbletea)
program with one tab per subpackage (`chat`, `models`, `usage`, `accounts`,
`coding`, `media`, `tools`). Each tab is an independent `tea.Model` with its
own `Update`/`View`; `pkg/tui/root.go` dispatches between them. Long-running
work (an API call, a filesystem operation) is wrapped in a `tea.Cmd` closure,
which Bubble Tea runs on its own goroutine — code called from a `tea.Cmd`
must not rely on package-level mutable state being uncontended (this bit us
once with `http.DefaultClient.Timeout`; see the fix in
`pkg/coding/validator.go`'s doc comment for the specifics).
