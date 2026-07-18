# Architecture

## Package layout

```
main.go               A five-line entrypoint: package main → internal/cli.Execute()
internal/cli/         CLI commands (package cli), one file per command group:
                      chat.go, accounts_cli.go, coding_cli.go, ...
pkg/client/           The Go library — one file per API service, no CLI/TUI dependency
internal/accounts/    Multi-account credential store (~/.config/zai-client/accounts.json)
internal/coding/      GLM Coding Plan credential store + per-tool config writers
internal/usageview/   Pure presentation helpers (time windows, heat maps, formatting) —
                      shared by both the CLI and the TUI so their output can never drift
internal/tui/         Bubble Tea terminal UI, one subpackage per tab
internal/fileinput/   FileOrURL: a URL passes through, a local path is base64-encoded —
                      shared by `ocr parse` and the TUI media tab
```

`pkg/client` has zero dependencies on anything CLI- or TUI-specific — it's
designed to be imported standalone (see the [Library Guide](library-guide.md)),
and is the **only** public package. Everything under `internal/` is
implementation the compiler forbids outside code from importing, so the CLI/TUI
layers can be refactored freely. The CLI and TUI are both thin callers of
`pkg/client`. `go install github.com/SamyRai/go-z-ai@latest` still builds the
root `main.go` into the `go-z-ai` binary.

## CLI conventions

Command handlers that need an API client are registered as
`RunE: runWithClient(runX)` and take the resolved `*client.Client` as a third
parameter — `runWithClient` (in `internal/cli/common.go`) resolves it once via
`getClient`, so the handlers don't each repeat the resolve-and-check preamble.
Credential precedence itself lives in `resolveConfig` (flag → `--account` →
`ZAI_API_KEY`/`KEY` → active account), unit-tested in `credentials_test.go`.

Output format is uniform: every command that prints a result registers the
shared `--format` flag via `addFormatFlag` and renders through `emit(cmd, v,
textFn)`, which emits pretty JSON for `--format json` and otherwise runs the
human-readable `textFn`. Progress chatter on JSON-capable commands goes to
stderr so stdout stays valid JSON.

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
| Agents | `https://api.z.ai/api` (bare root, no `/paas/v4`) by default; `https://open.bigmodel.cn/api` under `Config.Region = RegionChina` | Verified live — nesting `/v1/agents` under the chat-completions base 404s. |
| Everything else | `Config.BaseURL` (`/api/paas/v4`, or `/api/coding/paas/v4` for GLM Coding Plan accounts) | The general case. |

### Regional gateway selection (Config.Region)

Z.AI serves the same GLM model family from two regional gateways: the
international host `api.z.ai` and the China-mainland mirror
`open.bigmodel.cn`. Most services pick their host via `Config.BaseURL` (chat,
files, tools, etc.) or a fixed constant (Embeddings/Moderations always use the
China host). Four services — **monitor** (quota/usage), **biz** (account
info), **agents**, and **account-type detection** — used to be hardcoded to
`api.z.ai`, which left a `glm_coding_plan_china` key unable to reach its own
region's monitor/usage endpoints (and got mis-classified by
`accounts add`/`account detect`).

`Config.Region` (`RegionGlobal`, the default, or `RegionChina`) selects the
host for those four region-scoped services only. It does **not** override
`Config.BaseURL` (the chat surface) or the Embeddings/Moderations host.
From the CLI, use `--region {global,china}` or `ZAI_REGION` (aliases: `cn`,
`bigmodel`, `west`). An unknown value falls back to global rather than
erroring, so a typo never blocks an unrelated command.

The China mirror hosts for monitor/biz/agents/detection are modeled by
mirroring the `api.z.ai` path layout on `open.bigmodel.cn` and marked
`NOT VERIFIED LIVE` — the China platform is live-verified to serve the same
OpenAPI surface for `/models` and `/chat/completions` (see `BigModelBaseURL`),
but the monitor/biz/agents hosts on the China side have not been captured by a
cassette yet. Pin them with `ZAI_RECORD=1` if you hold an entitled China key.

## The live-verification convention

Z.AI's own SDKs and docs sometimes disagree with each other, and sometimes
with what the live API actually returns (an endpoint documented as optional
that 400s without it; an error embedded in a 200 response body; a field typed
differently across two official SDKs). Rather than trust a single source,
new services here are checked against a real API call and the interaction is
recorded as a [go-vcr](https://github.com/dnaeon/go-vcr) cassette
(`pkg/client/testdata/cassettes/`), replayed in `ModeReplayOnly` so the test
suite never touches the network.

Two files in `pkg/client` carry the live-verification work, with distinct
roles: **`live_replay_test.go`** holds the `Test*Live` tests that replay the
committed cassettes as frozen findings (entitlement gates, the
200-with-embedded-failure quirk, the single-key-across-hosts claim); it is the
running log of what's been confirmed and why it mattered. **`live_verify_test.go`**
holds the `TestVerify*` recording harness — each test SKIPS until you capture a
new success-path cassette with `ZAI_RECORD=1`, then replays it; it's the
to-do list of shapes still pending a real capture.

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
other) and for account-type-to-endpoint resolution (`internal/coding/plans.go`).
Prefer this shape over adding another conditional branch when you're adding a
new recognized value to an existing concept.

## Credential file safety

Both `internal/accounts` (the multi-account store) and `internal/coding` (the GLM
Coding Plan credential store, plus every third-party tool config it writes —
Claude Code, OpenCode, Crush, Factory Droid, Cursor) write via a temp-file
atomic write followed by rename, never a direct in-place write. A crash or
kill mid-write leaves the original file untouched instead of truncated —
important here specifically because several of these are *other programs'*
real config files being merged into, not files this project owns outright.
Files containing a key are created `0600`; directories `0700`.

## The TUI

`internal/tui` is a [Bubble Tea](https://github.com/charmbracelet/bubbletea)
program with one tab per subpackage (`chat`, `models`, `usage`, `accounts`,
`coding`, `media`, `tools`). Each tab is an independent `tea.Model` with its
own `Update`/`View`; `internal/tui/root.go` dispatches between them. Long-running
work (an API call, a filesystem operation) is wrapped in a `tea.Cmd` closure,
which Bubble Tea runs on its own goroutine — code called from a `tea.Cmd`
must not rely on package-level mutable state being uncontended (this bit us
once with `http.DefaultClient.Timeout`; see the fix in
`internal/coding/validator.go`'s doc comment for the specifics).
