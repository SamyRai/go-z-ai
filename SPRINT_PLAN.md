# Sprint Plan — Better Z.AI API support in `zai-client`

> Planned 2026-07-08. **Sprint A delivered 2026-07-08. Phase 3 (provider
> cleanup + media services + TUI) delivered 2026-07-10.** Operational
> checklist: [`todo.md`](./todo.md). API facts reconciled against the live
> `https://docs.z.ai` (quick-start + nav) and the in-repo client source.

## Sprint A status: ✓ delivered

| Item | Status | Validation |
|---|---|---|
| A1 Streaming | ✓ | 6 httptest SSE cases; live auth/transport confirmed |
| A2 Structured output | ✓ | marshal test; `--json-schema` CLI |
| A3 Function-calling loop | ✓ | 4 httptest cases incl. wire-format |
| A4 Thinking + advanced flags | ✓ | CLI wired |
| A5 Retry/backoff | ✓ | 8 httptest cases; live non-retry-on-quota observed |

Full repo `go build` / `go vet` / `go test` clean. The active account's quota
window was exhausted during live validation, so a 200 happy-path stream wasn't
observable — it's covered by the SSE test suite. See `todo.md` "Known limitations".

## Context

`zai-client` is a Go CLI + `pkg/client` SDK for Z.AI. Two review passes informed
this plan:

1. **Live API** — `docs.z.ai` lists language models (GLM-4.6 / GLM-4.5 /
   GLM-4-32B-0414-128K), vision (GLM-4.6V / GLM-4.5V), image gen (CogView-4),
   video gen (CogVideoX-3, Vidu Q1, Vidu 2), capabilities (Deep Thinking,
   Streaming, Tool Streaming, Function Calling, Context Caching, Structured
   Output), tools (Web Search), agents, and the general (`/api/paas/v4`) vs
   coding-plan (`/api/coding/paas/v4`) endpoints.
2. **Codebase** — `pkg/client` ships chat, models, usage, quota, accounts,
   tools, detection services with structured errors + tests. The account /
   usage / quota surface (multi-account store, coding-plan auto-detection,
   14-day usage heat map) is strong; the **chat completions** surface is thin.

## Coding-helper port: ✓ delivered (separate workstream)

A faithful Go port of the official `npx @z_ai/coding-helper` (read from its npm
source, v0.0.7) — credential management + coding-tool configuration. Not part of
Sprint A/B; tracked under its own `C1–C8` items in `todo.md`.

- `pkg/coding`: plans, `~/.chelper/config.yaml` store (byte-compatible with the
  Node helper), `/models` key validation, tool registry, and per-tool
  Load/Unload/Detect for **Claude Code, OpenCode, Crush, Factory Droid**.
- `coding` CLI: `auth / load / unload / status / tools / doctor`.
- **Goes beyond the official helper for Claude Code**: optional
  `ANTHROPIC_DEFAULT_*_MODEL` mapping (Z.AI's documented defaults) +
  `CLAUDE_CODE_AUTO_COMPACT_WINDOW` + thinking/output budgets; `--no-model-mapping`
  reproduces the bare helper format.
- Fixed a real bug: `provider enable-claude` now writes `ANTHROPIC_AUTH_TOKEN`
  (was `ANTHROPIC_API_KEY`); Z.AI's endpoint authenticates via `Bearer`.

## Current state (what's solid)

- Multi-account management with `coding_plan` / `pay_as_you_go` auto-detection
  via a free probe (`pkg/accounts`).
- Quota monitoring across all accounts; token + tool-usage time series
  (`pkg/client/quota.go`).
- Structured error parsing with tests (`errors.go`, `errors_test.go`).
- Layered config: flags > env > `.env` > defaults.
- `ChatRequest` already carries typed fields for `Tools`, `ToolChoice`,
  `ResponseFormat`, `Thinking`, `Stop` — so most chat work is wiring, not
  new types.

## Gap analysis

| Capability | Live API | Codebase | Status |
|---|---|---|---|
| Streaming chat | yes (SSE) | `CreateStream()` is a stub | ❌ |
| Function/tool calling | yes | typed, no CLI, no loop helper | ⚠️ types only |
| Structured output | yes (`json_schema`) | `ResponseFormat` = text/json_object only | ❌ |
| Deep Thinking | yes | typed, not in CLI | ⚠️ types only |
| Vision / multimodal | GLM-4.6V / 4.5V | `Message.Content` is `string` only | ❌ |
| Built-in `web_search` in chat | yes | `Tool.Type` documents `function` only | ❌ |
| Context caching | yes | absent | ❌ |
| Image generation | CogView-4 / GLM-Image | absent | ❌ |
| Video generation | CogVideoX-3 / Vidu | absent | ❌ |
| Agents (Slide/Poster, Translation) | yes (beta) | absent | ❌ |
| Retry / backoff | — | single-attempt, no 429/5xx retry | ❌ |
| Model catalog freshness | GLM-5.2 live; docs nav lags at 4.6 | hardcoded in a doc, not from `/models` | ⚠️ |

## Bugs / smells

- `pkg/client/tools.go` — each method builds its own `http.Client{30s}`,
  bypassing `Config.Timeout` and `parseAPIError`; endpoints (`/api/tools/*`)
  don't match the tool codes tracked in `quota.go` (`search-prime` /
  `web-reader` / `zread`). Partly speculative.
- `chat.go` CLI — `--stream` advertises "not yet implemented"; thinking /
  tools / json_schema / stop / image are not exposed.
- `ResponseFormat` lacks `json_schema`.
- No multimodal message content.
- README architecture diagram shows a `cmd/` tree that doesn't exist (flat repo).
- `client.go` defines `AnthropicBaseURL` (`/api/anthropic`) but it is unused.

## Sprint A — chat completions surface (highest ROI)

Prerequisites first: **A1 streaming** and **A5 retry** de-risk everything after.

1. **A1 · Streaming (L)** — real SSE in `pkg/client`, CLI `--stream`.
2. **A2 · Structured output (S)** — `json_schema` in `ResponseFormat`, CLI flag.
3. **A3 · Function-calling loop (M)** — `RunWithTools` helper + CLI `--tool`.
4. **A4 · Thinking + advanced flags (S)** — wire existing typed fields into CLI.
5. **A5 · Retry / backoff (M)** — wrap `doRequest`, configurable, reusable.

**Done when** the CLI can stream, request a json_schema, run a tool-calling
loop, and set thinking/effort — all while retrying transient failures.

## Sprint B — generation, multimodal, correctness

6. **B1 · Multimodal messages (M)** — backward-compatible `Content` type, `--image`. **← open, moved to Sprint C**
7. **B2 · Image generation service (M)** — `images.go`, CogView-4 / GLM-Image. **✓ Phase 3**
8. **B3 · Video generation service (M)** — `videos.go`, async submit + poll. **✓ Phase 3**
9. **B4 · Fix `tools.go` (S)** — route through `httpClient`, verify endpoints. **← open, moved to Sprint C**
10. **B5 · Tests + catalog hardening (M)** — golden tests; drive `/models` live. **← open, split across Sprint C/D**

## Phase 3 status: ✓ delivered (2026-07-10, uncommitted)

Superset of B2/B3 plus structural cleanup — see `todo.md` progress log:

- Deleted the fictional `pkg/provider` / `pkg/appconfig` / `provider_cli.go`
  abstraction; Cursor folded into `pkg/coding` as the 5th tool.
- New services: `images.go` (sync+async), `videos.go`, `audio.go` (multipart),
  `layout.go` (OCR), `async.go` (shared poller). CLI `image/video/audio/ocr`
  commands; TUI rebuilt as `pkg/tui/*` with a Media tab.
- `go build` / `go vet` / `go test -race` clean; 7-tab pty smoke test passed.
- **Debt created:** the five new service files have zero tests; new methods
  don't take `context.Context`; endpoints not yet validated live.

## Sprint C — commit, correctness, multimodal, coverage

**Status (2026-07-10): C0–C5 delivered; C6 pending (user-gated, spends real
quota).** See `todo.md` for full detail per item; commits: C0 (5 batches,
prior session), C1 `72c80ea`, C2 (same commit as C1), C4 `ffa1c42`, C5
`99c124c`, C3 `36fae1a`. Full green gate (`go build`/`vet`/`test -race`/
`gofmt`/`govulncheck`) after every commit.

Review verdict (2026-07-10): the codebase is structurally healthy (largest
file 572 LOC, packages own real domains, race-clean) but carries risk in
three places: **nothing is committed**, the newest surface is **untested**,
and `tools.go` is the last service bypassing the hardened transport.
Performance: no measured bottleneck exists and this is a CLI/TUI without hot
paths — no optimization work is justified; the one perf-adjacent fix
(per-call `http.Client` in `tools.go` defeating connection reuse) is C2.

Priority order (value ÷ effort, dependencies respected):

1. **C0 · Commit + repo hygiene (S)** ✓ — batch conventional commits of all
   in-flight work; removed `usage.go.backup` and stray built binaries;
   `.gitignore` covers them.
2. **C1 · Context plumbing (M)** ✓ — every `pkg/client` service method now
   takes `ctx context.Context` first. Core: `doRequest`/new `doRequestBase`
   are ctx-first; ~25 call sites updated (CLI via `cmd.Context()`, TUI via
   `context.Background()` — no cancel plumbing yet).
3. **C2 · Fix transport-bypassing services (S → scope grew)** ✓ — `tools.go`
   was the known offender; auditing every C1 call site found the identical
   bug (private `http.Client{30s}`, no retry, no `parseAPIError`) already
   present in `account.go` and `quota.go` too. Fixed all three via
   `doRequestBase` + a new `ToolsBaseURL` const. Endpoints unchanged, still
   unverified live (C6).
4. **C3 · Multimodal messages (M)** ✓ — `Message` gained `Images []string`;
   `Content` stays a plain `string` (zero source-compat break). Custom
   `MarshalJSON`/`UnmarshalJSON` switch the wire shape only when `Images` is
   set. CLI `chat create --image url|@path` (repeatable). TUI attachment
   stays out of scope (Sprint D).
5. **C4 · Media-service tests + poll helper (M)** ✓ — 15 httptest cases
   across images/videos/layout/audio/async; added
   `Client.WaitForResult(ctx, id, interval)`.
6. **C5 · Streaming timeout (S)** ✓ — root cause was `http.Client.Timeout`
   bounding the whole request *including* streamed-body reads; fixed by
   moving the timeout to transport-level dial/handshake/header-wait instead.
   No API change needed.
7. **C6 · Live media smoke test (S, user-gated)** — pending. One cheap real
   call per new endpoint to confirm response shapes; spends real quota —
   run only with explicit go-ahead from the account owner.

**Done when** all in-flight work is committed in reviewable batches, every
`pkg/client` service goes through the shared transport with a caller
context, vision chat works from the CLI, and the media surface has the same
httptest coverage standard as chat/errors.

```
C0 (commit) ─► C1 (ctx signatures) ─► C2, C4
                                       C3 (independent) ─► TUI image attach (D)
                                       C5, C6 (anytime after C0)
```

## Sprint D — candidates (next)

- Model catalog from live `/models` + GLM-5.x reconciliation (B5 remainder).
- CLI tool auto-exec loop (needs a security-gated tool→command convention).
- TUI automated tests (`teatest`): tab switching, form submit, quit restore.
- Root-level CLI refactor: `usage.go` (385 LOC) and `accounts_cli.go`
  (545 LOC) are the two largest non-package files; split by command surface.

## SDK/CLI parity research (2026-07-10)

Researched `docs.z.ai` and the official `zai-org/z-ai-sdk-python` (the
current SDK; supersedes legacy `zhipuai`/`MetaGLM/zhipuai-sdk-python-v4`)
to find real capability gaps. **There is no official Z.AI CLI** — the only
"zai-cli" on npm is third-party (`numman-ali/zai-cli`), not a parity
target. Full detail with endpoint shapes and source links in `todo.md`'s
"Sprint D candidates" section; headline findings:

- **Already covered, nothing to do**: Context Caching is fully automatic
  server-side and `Usage.PromptTokensDetails.CachedTokens` already surfaces
  it; tool-calls-in-stream is already handled.
- **Real gaps, no CLI/SDK coverage at all**: Assistant API (a second,
  distinct conversation API alongside the already-backlogged Agents API —
  unclear which is authoritative, verify live before picking one), Batch
  API (needs Files API first — batch input is an uploaded JSONL file),
  Files API, Moderations API, Handwriting OCR (distinct endpoint from the
  layout-parsing/glm-ocr we already have), Voice cloning API.
- **Verify-before-building caveats**: Embeddings has an open upstream issue
  ([#67](https://github.com/zai-org/z-ai-sdk-python/issues/67)) reporting
  it doesn't work on the global `api.z.ai` endpoint; Voice cloning isn't in
  `docs.z.ai`'s own doc index at all (may be beta/region-restricted).

## Later / backlog (unchanged)

Anthropic-protocol wrapper (`/api/anthropic`), request/response logging +
metrics, benchmarks + `benchstat` CI gate — benchmarks only once an actual
hot path exists. Agents/Batch/Assistant/Files/Embeddings/Moderations/Voice
now scoped in detail above and in `todo.md` rather than one-line bullets.

## Sequencing & dependencies

```
A5 (retry) ──┐
A1 (stream) ─┼─► A3 (tool loop, needs stream for tool streaming) ─► B1..B3
A2 (schema) ─┘                                              A4 (independent, anytime)
                                                            B4, B5 (cleanup, anytime)
```

A1 and A5 are the load-bearing items; do them first. A2/A3/A4 are mostly
wiring of already-typed fields. B-sprint items are independent and can be
parallelized. Items that hit unverified endpoints (B2/B3) must confirm paths
against live docs before merge.

## Open question (decide before sizing B)

**Is `zai-client` primarily (a) a personal account/usage CLI or (b) a
general-purpose Go SDK?** It is currently strong at (a) and thin at (b).
- If (a): Sprint A's CLI wiring + retry is essentially the whole sprint; trim B.
- If (b): promote B1 (multimodal) + B2/B3 (generation) up; the usage UI work
  is maintenance-only.

## Notes for implementation

- Z.AI is OpenAI-compatible: SSE deltas (`choices[].delta`), `[DONE]`,
  incremental `tool_calls[].function.arguments`, `image_url` content parts,
  `/images/generations`. Verify each against `docs.z.ai` before relying on it.
- Keep exported `ChatRequest` / `Message` shapes backward-compatible; new
  behavior is additive (new fields, new optional CLI flags).
- End each item with a golden test (B5 makes this systematic) and a README
  example.
