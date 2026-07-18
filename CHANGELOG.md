# Changelog

Notable changes to this project, loosely following
[Keep a Changelog](https://keepachangelog.com/). This project doesn't cut
version tags yet — entries are grouped by date.

## 2026-07-18

### Added
- **Regional gateway selection (`Config.Region` / `--region` / `ZAI_REGION`).**
  Z.AI serves the same GLM model family from two regional gateways: the
  international host `api.z.ai` (the historical default) and the China mirror
  `open.bigmodel.cn`. Previously only Embeddings/Moderations were wired to the
  China host; monitor (quota/usage), biz (account info), agents, and account
  detection were hardcoded to `api.z.ai`, so a `glm_coding_plan_china` key
  couldn't reach its own region's usage/account endpoints and got
  mis-classified by `accounts add`/`account detect`. `Config.Region`
  (`RegionGlobal`, the default, or `RegionChina`) now selects the host for
  monitor, biz, agents, and detection. From the CLI: `--region {global,china}`
  or `ZAI_REGION` env (aliases `cn`, `bigmodel`, `west`); an unknown value
  falls back to global rather than erroring. `internal/coding/plans.go`
  mirrors this with `MonitorBaseURL` / `BizBaseURL` / `AgentsBaseURL` plan
  helpers. The China hosts are `NOT VERIFIED LIVE` (modeled by mirroring the
  `api.z.ai` path layout on `open.bigmodel.cn`, which is live-verified for
  `/models` and `/chat/completions`). See `docs/architecture.md`.
- **Chat-completion API sync (`pkg/client/types.go`, `chat.go`).** New fields
  matching the current docs.z.ai chat-completion spec, all additive and
  `NOT VERIFIED LIVE` until a cassette pins them:
  - `ChatRequest.StreamToolCall` — streamed tool-call deltas (GLM-4.6+).
  - `Tool` now discriminates across `function` / `retrieval` / `web_search`
    types via `NewFunctionTool` / `NewRetrievalTool` / `NewWebSearchTool`.
    The `web_search` payload shape (`{"search_query":[...]}`) follows the
    official Python SDK example; `retrieval` and the full tool-type support
    are unverified live.
  - `ChatResponse.WebSearch` — the top-level `web_search` array returned when
    a web_search tool fires (entry shape reuses `WebSearchResult` from
    `tools.go`).
  - `ThinkingConfig.Effort` now documents `xhigh` (GLM-5.2; `xhigh`→`max`).
    `validateChatRequest` rejects unknown effort values client-side.
  - `FinishReason*` constants for the live values `sensitive`,
    `model_context_window_exceeded`, `network_error`.
  - Client-side tool-name guard (`^[A-Za-z0-9_-]{1,64}$`), a 128-function
    cap, and per-type payload validation (a `function`/`retrieval`/`web_search`
    tool must carry its matching payload; unknown types are rejected).
- **Live-verification scaffolding** for four more services in
  `pkg/client/live_verify_test.go`: Voice Clone, Voice Delete,
  Chat-Stream-Tool-Call, and Chat-Web-Search-Response. Each skips until a
  cassette exists (CI stays green); record one with
  `ZAI_RECORD=1 ZAI_API_KEY=<key> go test -run TestVerify<Name> ./pkg/client`
  (the harness redacts `Authorization` to `Bearer REDACTED` before saving).

### Docs
- Aligned every doc with the 2026-07-18 sprint: `--region`/`ZAI_REGION` added
  to the CLI Reference global-flags table; `xhigh` added to the `--effort`
  values; `Config.Region` added to the Library Guide Config table; the new
  tool types (`NewRetrievalTool`/`NewWebSearchTool`), `StreamToolCall`,
  `ChatResponse.WebSearch`, the `FinishReason*` constants, the 128-function
  cap, and the tool-name regex documented in the Library Guide's Function
  calling section; Accounts & Quota's "China platform key" rewritten as
  "Regional gateways" covering both the Embeddings/Moderations axis and the
  monitor/biz/agents/detection axis; Getting Started's China note reframed;
  Architecture's "three services" corrected to four (detection also went
  region-aware).
- Rewrote `docs/roadmap.md` as a crisp, current task list — every
  "Unverified live" item names its exact `TestVerify*` test + recording
  command, grounded in the cassette inventory (no item was removed: none of
  the unverified services got a success cassette this sprint).
- Renamed `pkg/client/live_verification_test.go` → `live_replay_test.go` so
  the two live-test files self-document: `live_replay_test.go` holds the
  `Test*Live` replay-only tests (frozen findings), `live_verify_test.go`
  holds the `TestVerify*` recording harness. No behavior change.
- Index completeness: `docs/README.md` now lists `.github/SETUP.md` and
  `.env.example`; root `README.md`'s doc table now matches the docs/ index
  (Roadmap, Security, Changelog rows added).
- `.github/PULL_REQUEST_TEMPLATE.md` checklist aligned with CONTRIBUTING.md
  (added `golangci-lint run` and `govulncheck`).

## 2026-07-17

### Changed
- **Repository layout (breaking for importers of the app packages).** The CLI
  command code moved from `package main` at the repo root into `internal/cli`
  (a five-line root `main.go` now just calls `cli.Execute()`), and the
  in-repo-only packages `pkg/tui`, `pkg/usageview`, `pkg/accounts`, and
  `pkg/coding` moved under `internal/`. Only `pkg/client` remains a public,
  importable package — the documented library surface is now compiler-enforced.
  `go install github.com/SamyRai/go-z-ai@latest` still produces the `go-z-ai`
  binary; CLI behavior and `--help` output are unchanged. If you imported
  `pkg/accounts` or `pkg/coding` directly (previously documented as reusable),
  that import path no longer exists — drive the functionality through the
  `accounts`/`coding` CLI commands instead.
- `interface{}` → `any` throughout (mechanical; `any` is an alias, so
  `pkg/client`'s exported signatures are unchanged for consumers).

### Added
- Consistent `--format text|json` on every result-producing command. `batch`,
  `files`, `image`, `video`, `rerank`, and `ocr` gained JSON output (were
  text-only); `embeddings`/`moderations` keep JSON as the default but gained a
  text summary. Progress messages on JSON-capable commands now go to stderr so
  stdout stays valid JSON.
- First tests for the CLI layer: credential-precedence coverage
  (`resolveConfig`), an end-to-end cobra harness against `httptest`, and unit
  tests for `buildChatRequest` and the new `internal/fileinput` helper.

### Internal
- `runWithClient` wrapper replaces the four-line `getClient` preamble repeated
  across ~50 command handlers. `addFormatFlag`/`emit` helpers centralize output
  formatting. `internal/fileinput.FileOrURL` de-duplicates the OCR file/URL
  handling previously copied between `ocr parse` and the TUI media tab.

### Fixed
- `accounts list/show/current --format json` masked the API key like the table
  view does (it previously printed the raw key); `--reveal` opts into raw keys
  for export/backup.
- `usage`/account status now correctly reports an insufficient-balance account
  as *accessible but out of balance* instead of *inaccessible*. It classifies
  the failure from the structured `*APIError` (code/HTTP status) rather than
  string-matching a message; distinguishes 401 (auth) and 429 (rate limit) too.
- TUI: submitting a media job (esp. a multi-minute video) and switching tabs no
  longer strands the result — async results are routed back to the originating
  tab, with esc-to-cancel.

### Testing / docs
- First unit tests for the credential store (`internal/accounts`), every TUI
  tab, and the CLI credential-precedence path. New opt-in live-verification
  harness (`ZAI_RECORD=1`) records redacted cassettes for previously docs-only
  success shapes (Anthropic Messages, Embeddings, Moderations, Agents). Noted
  the vision + tool-calling 401 pitfall in the CLI reference.

## 2026-07-12

### Added
- Quota burn-rate ("Pace") indicator on token windows in `accounts quota` /
  `usage quota` and the TUI Usage tab: extrapolates each rolling window's own
  reported usage against elapsed window time to flag when you're on pace to run
  out before reset (`62% used at 55% of window elapsed — on pace to run out
  ~24m before reset`). Straight-line math on real API fields — no peak/off-peak
  pricing assumptions. New `QuotaLimit.WindowDuration()`/`WindowStart()` and
  `usageview.Pace`/`FormatPace` (the first tests for the previously
  untested `usageview` package). Directly targets the common "limits run out
  sooner than expected" complaint.
- Anthropic-compatible Messages client (`AnthropicService`, `c.Anthropic()`) —
  a typed Go client for Z.AI's `/api/anthropic` surface (`POST /v1/messages`),
  the endpoint the GLM Coding Plan points Claude Code at, parallel to the
  OpenAI-style `Chat` service. Covers `Create`, streaming `CreateStream` (raw
  Anthropic SSE events), text/image/tool_use/tool_result content blocks, tools
  (with the same schema-compat rewrite), and Bearer auth + `anthropic-version`
  header. New CLI: `anthropic messages <prompt> [--stream ...]`. Routing/auth
  are confirmed reaching the live endpoint (bogus key → clean HTTP 401); the
  success-path body shape is documented, not yet live-verified (see
  [Roadmap](docs/roadmap.md)).
  - Extended thinking: `AnthropicThinking` request config, `thinking`/
    `redacted_thinking` response blocks, and `resp.Thinking()` — which falls
    back to an OpenAI-style `reasoning_content` field if GLM surfaces reasoning
    that way instead of as a thinking block (the claude-code-router#1133 case).
    CLI `--thinking-budget N` enables it and prints reasoning to stderr.
- Tool-schema compatibility: chat requests now normalize tool (function)
  `parameters` into the flat JSON-Schema subset GLM's parser accepts, instead
  of letting `anyOf`/`oneOf`/`allOf`/`$ref`/`$defs` reach the endpoint and come
  back as an opaque HTTP 500 (a pain point for tools generated from typed
  languages — nullable fields, reused structs, composed types). Nullable
  unions collapse to their underlying type, `allOf` merges, and local `$ref`s
  inline (with cycle protection). Exposed as `client.SanitizeToolSchemas` for
  explicit use, applied automatically before every chat request, and disablable
  via `Config.DisableToolSchemaCompat`. See `pkg/client/toolschema.go` and
  [Library Guide](docs/library-guide.md#tool-schema-compatibility). The exact
  set of server-rejected constructs is drawn from community reports, not yet
  reproduced live here (see [Roadmap](docs/roadmap.md)).

### Changed
- golangci-lint is now part of the gate: a checked-in `.golangci.yml` (default
  linter set — errcheck, govet, ineffassign, staticcheck, unused), a
  `golangci-lint` CI job on every push/PR, and a line in the CONTRIBUTING
  pre-PR checklist. The config deliberately keeps `io.Reader.Read` checked so
  the short-read pattern below can't come back unnoticed.

### Fixed
- Short-read bug in test HTTP servers — a single `r.Body.Read` into a
  `ContentLength`-sized buffer (`Read` isn't guaranteed to fill the buffer in
  one call, so body assertions could flake). A first pass fixed four files;
  golangci-lint then surfaced eight more occurrences across the moderations,
  rerank, tools, voice, and layout tests, now all on `io.ReadAll`.
- `staticcheck` SA9003 empty `if` branch in `main.go`'s config load, collapsed
  to the same `_ = ...` idiom already used for the `.env` load above it.

### Added
- `coding mcp add/remove/status`: registers Z.AI's official Vision MCP Server
  (`@z_ai/mcp-server` — screenshot OCR, error-screenshot diagnosis,
  diagram/chart understanding, image/video analysis via GLM-4.6V) into any of
  the five supported coding tools, matching the "manage MCP services" step of
  the official `@z_ai/coding-helper` wizard that this client otherwise ports
  in full. Each tool gets its correct file and JSON shape — notably, Claude
  Code and Factory Droid keep MCP config in a different file than their
  provider/credential config. Available from the CLI and the TUI's Coding tab
  (`m` key).

## 2026-07-11

### Added
- Agents service (`Invoke`, `AsyncResult`) — live-verified, including the
  200-with-embedded-business-failure response quirk both endpoints share.
- Embeddings, Moderations, Rerank, Voice (cloning), and FileParser services
  and CLI commands.
- Handwriting OCR (`ocr handwriting`), distinct from layout parsing (`ocr parse`).
- A [go-vcr](https://github.com/dnaeon/go-vcr)-based live-verification test
  suite (now `pkg/client/live_replay_test.go` + `live_verify_test.go`,
  `testdata/cassettes/`) that replays real recorded API interactions instead
  of hand-written fixtures.
- Cursor as a fifth supported coding tool alongside Claude Code, OpenCode,
  Crush, and Factory Droid.
- Full documentation rewrite: a `docs/` guide (Getting Started, CLI
  Reference, Accounts & Quota, Coding Tools, Library Guide, Error Handling,
  Architecture), `CONTRIBUTING.md`, `SECURITY.md`, this changelog, and a CI
  workflow (build/vet/gofmt/test -race/govulncheck).

### Changed
- Module path renamed to `github.com/SamyRai/go-z-ai` (was the
  non-installable `zai-api-client`) ahead of the public release.
- Licensed under Apache 2.0.
- `pkg/coding`'s API-key validator no longer mutates `http.DefaultClient`
  (a shared global) — it now bounds the request with `context.Context`
  instead, fixing a data race under concurrent callers (the TUI validates
  keys from a background goroutine).
- Config-file writers in `pkg/coding` (the credential store and every
  third-party tool config it edits) now write atomically via
  temp-file-then-rename, matching the pattern `pkg/accounts` already used —
  a crash mid-write can no longer truncate your Claude Code/OpenCode/Crush/
  Factory Droid/Cursor settings.
- Resource IDs (`batchID`, `fileID`, task IDs) are now URL-path-escaped
  before being interpolated into request paths.
- Removed unused "legacy compatibility" error constructors/sentinels from
  `pkg/client` that had no real callers anywhere in the codebase.

## 2026-07-10

### Added
- Streaming chat completions (SSE), with CLI `chat create --stream`.
- Structured output (`json_schema` response format, `--json-schema`).
- Function-calling: a `RunWithTools`/`RunWithToolsLimit` auto-executing loop,
  plus CLI tool declarations (`--tool`).
- Deep-thinking controls (`--thinking`, `--effort`) and advanced sampling
  flags (`--stop`, `--top-p`, `--do-sample`, `--show-reasoning`).
- Automatic retry with exponential backoff, jitter, and `Retry-After`
  support on 429/5xx/network errors.
- Multimodal messages (`Message.Images`) for vision models (GLM-4.6V/4.5V),
  wire-compatible with plain-text messages when no image is attached.
- Image generation (`glm-image`/CogView-4), video generation
  (CogVideoX-3/Vidu, always async), audio transcription/TTS, and OCR
  (layout parsing) services and CLI commands.
- Files and Batch API services for bulk/async request processing.
- A full-screen terminal UI (`zai-client tui`) with seven tabs: chat, models,
  usage, accounts, coding, media, tools.
- `pkg/usageview`, a presentation-only package shared by the CLI and TUI so
  usage/quota rendering (time windows, heat maps, relative timestamps)
  can't drift between the two.

### Changed
- `context.Context` is now the first parameter on every `pkg/client` service
  method, threaded all the way to the HTTP call.
- Removed a provider/app-config abstraction layer in favor of the simpler
  multi-account model.

### Fixed
- `CreateStream` no longer gets cut off mid-generation by `Config.Timeout` —
  the timeout now bounds dial/TLS/response-header wait, not the whole
  response body read.
- `Tools`, `Account`, and `Quota` services no longer built their own
  unconfigured `http.Client` per call (bypassing retry, timeout, and
  structured error parsing) — routed through the shared request facade.

## 2026-07-08

### Added
- Initial release: chat completions, models, usage/quota/billing monitoring,
  and account operations, as both a CLI and a Go client library.
- A Go port of `@z_ai/coding-helper` (`pkg/coding`) for configuring Claude
  Code, OpenCode, Crush, and Factory Droid to use a GLM Coding Plan
  credential, sharing the official helper's `~/.chelper/config.yaml` file.
- Multi-account credential management (`pkg/accounts`) with automatic
  `coding_plan`/`pay_as_you_go` type detection.
- Structured API error parsing with categories, user-facing messages, and
  retriable flags.
