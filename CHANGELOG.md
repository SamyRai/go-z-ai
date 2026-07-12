# Changelog

Notable changes to this project, loosely following
[Keep a Changelog](https://keepachangelog.com/). This project doesn't cut
version tags yet — entries are grouped by date.

## 2026-07-12

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
  suite (`pkg/client/live_verification_test.go`, `testdata/cassettes/`) that
  replays real recorded API interactions instead of hand-written fixtures.
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
