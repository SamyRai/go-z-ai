# TODO — zai-client

Operational work tracker. Rationale and gap analysis live in
[`SPRINT_PLAN.md`](./SPRINT_PLAN.md); this file is the checklist.

**Status:** `[ ]` pending · `[~]` in progress · `[x]` done · `[!]` blocked

**Progress log**
- 2026-07-08 — Sprint A complete (A1–A5). All `pkg/client` tests green; full
  repo `go build`/`go vet`/`go test` clean. Live integration confirmed against
  z.ai (auth + transport + structured-error + non-retry-on-quota all observed);
  happy-path 200 streaming proven by the httptest SSE suite — the active
  account's quota window was exhausted at validation time.
- 2026-07-08 — `@z_ai/coding-helper` port (`pkg/coding` + `coding` CLI).
  Faithful Go port of the official Node helper's source (v0.0.7): credential
  store (`~/.chelper/config.yaml`, byte-compatible with chelper), plan/endpoint
  model, `/models` key validation, and per-tool config writers for Claude Code,
  OpenCode, Crush, Factory Droid. `provider enable-claude`/status rewired to it,
  fixing the ANTHROPIC_API_KEY→AUTH_TOKEN bug. Verified interop by reading the
  real chelper-written `config.yaml` + `~/.claude/settings.json`.
- 2026-07-10 — **Phase 3** delivered (uncommitted): deleted `pkg/provider` /
  `pkg/appconfig` / `provider_cli.go`; Cursor folded into `pkg/coding` (5th
  tool, isolated-`$HOME` round-trip test); new `pkg/client` services —
  `images.go` (CogView, sync+async), `videos.go` (async), `audio.go`
  (multipart transcription), `layout.go` (OCR), `async.go` (shared poller) +
  `Images()/Videos()/Audio()/Layout()` accessors; CLI `image/video/audio/ocr`
  commands; TUI rebuilt under `pkg/tui/*` with a Media tab replacing the
  Provider tab. `go build`/`go vet`/`go test -race` clean; 7-tab pty smoke
  test passed. Live endpoints NOT yet validated (costs quota). B2/B3 thereby
  delivered; B1/B4/B5 remain open → Sprint C.
- 2026-07-08 — Claude Code **enhancements** beyond the official helper:
  optional model mapping (`ANTHROPIC_DEFAULT_{HAIKU,SONNET,OPUS}_MODEL`,
  defaulting to Z.AI's documented glm-4.5-air/glm-4.7/glm-4.7) plus
  `CLAUDE_CODE_AUTO_COMPACT_WINDOW` (default 1000000) and optional
  `MAX_THINKING_TOKENS` / `CLAUDE_CODE_MAX_OUTPUT_TOKENS`. Applied by default on
  `coding load claude-code`; `--no-model-mapping` / `--auto-compact-window 0`
  reproduce the bare chelper format. 3 new tests; full repo green.


---

## Sprint A — Chat completions surface ✓

- [x] **A1 · Streaming** — `pkg/client/chat.go` `CreateStream(ctx, req, onChunk)`
  parses SSE (`data:`/`[DONE]`), delta-accumulates content/reasoning/tool_calls,
  connect-level retry, context-aware. CLI `chat create --stream`. Tests: 6 SSE cases.
- [x] **A2 · Structured output** — `ResponseFormat` extended with `json_schema`
  (`NewJSONSchemaFormat`); CLI `--json-schema @file|--schema-name|--schema-strict`. Tests: marshal.
- [x] **A3 · Function-calling loop** — `pkg/client/chat_tools.go`
  `RunWithTools`/`RunWithToolsLimit` (bounded rounds, executor errors recorded).
  `Message` extended with `tool_calls`/`tool_call_id`/`name` (backward-compatible).
  CLI `--tool @tools.json` declares tools (prints tool_calls; no CLI auto-exec).
  Tests: happy path wire-format, no-tools, executor-error, round-limit.
- [x] **A4 · Thinking + advanced flags** — CLI `--thinking`, `--effort`, `--stop`,
  `--top-p`, `--do-sample`, `--show-reasoning`.
- [x] **A5 · Retry/backoff** — `doRequest` retry loop (429/5xx/network),
  exponential backoff + jitter, `Retry-After`, `Config.MaxRetries`/`RetryDelay`
  (`-1` disables). Removed dead `createRequest`. Tests: 8 cases.

---

## Coding-helper port (@z_ai/coding-helper) ✓

A faithful Go port of the official `npx @z_ai/coding-helper` ("chelper"), read
from its npm source (v0.0.7). Lives in `pkg/coding`; surfaced via `coding` CLI.

- [x] **C1 · Credential store** — `~/.chelper/config.yaml` (lang/plan/api_key),
  YAML keys match chelper for interoperability; 0600 perms.
- [x] **C2 · Plans + endpoints** — `glm_coding_plan_global` (api.z.ai) /
  `glm_coding_plan_china` (open.bigmodel.cn); anthropic + coding base URLs.
- [x] **C3 · Key validation** — `GET /coding/paas/v4/models`, 401=invalid.
- [x] **C4 · Tool registry** — claude-code/opencode/crush/factory-droid with
  command/config-path/install-command; `IsInstalled` via `exec.LookPath`.
- [x] **C5 · Per-tool writers** — Load/Unload/Detect for all 4 tools, preserving
  unknown JSON keys (mirrors chelper's object spreads).
- [x] **C6 · CLI** — `coding auth/revoke/load/unload/status/tools/doctor`.
- [x] **C7 · Bug fix** — `provider enable-claude`/status now write/read the
  official `ANTHROPIC_AUTH_TOKEN` format (was `ANTHROPIC_API_KEY`+model mapping).
- [x] **C8 · Claude Code enhancements** — optional model mapping (Z.AI's
  documented haiku/sonnet/opus defaults), `CLAUDE_CODE_AUTO_COMPACT_WINDOW`
  (default 1M), `MAX_THINKING_TOKENS`, `CLAUDE_CODE_MAX_OUTPUT_TOKENS`. CLI flags
  `--haiku/--sonnet/--opus/--auto-compact-window/--max-thinking-tokens/
  --max-output-tokens/--no-model-mapping`. Plain `LoadClaudeCode` stays
  chelper-exact; `LoadClaudeCodeOpts(DefaultClaudeOptions())` adds the extras.
- [x] Tests: 19 cases (per-tool round-trips, store, validator httptest,
  claude tuning).

## Sprint B — Generation, multimodal, correctness (closed 2026-07-10)

- [ ] **B1 · Multimodal messages** — → **C3**.
- [x] **B2 · Image generation service** — `pkg/client/images.go` (sync + async);
  CLI `image generate/status`; TUI Media tab. Delivered in Phase 3 (no tests yet → C4).
- [x] **B3 · Video generation service** — `pkg/client/videos.go`, async submit +
  shared `async.go` poller; CLI `video generate/status`. Delivered in Phase 3
  (no tests yet → C4).
- [ ] **B4 · Fix `pkg/client/tools.go`** — → **C2**.
- [ ] **B5 · Tests + catalog hardening** — media-service tests → **C4**; live
  `/models` catalog + GLM-5.x reconciliation → Sprint D.

## Sprint C — Commit, correctness, multimodal, coverage (planned 2026-07-10)

Rationale and sequencing in [`SPRINT_PLAN.md`](./SPRINT_PLAN.md).

- [ ] **C0 · Commit + repo hygiene** — batch conventional commits of ALL
  in-flight work (Sprint A + coding port + Phase 3 sit uncommitted on top of
  the single initial commit); delete `usage.go.backup`; remove + `.gitignore`
  built binaries (`zai-client`, `api-explorer/zai-client`,
  `playground/playground`).
- [x] **C1 · Context plumbing** — `context.Context` is now the first param on
  every `pkg/client` service method. Core change: `client.go`'s `doRequest`
  is ctx-first; added `doRequestBase(ctx, baseURL, ...)` so services hitting
  a non-default base URL (monitor/biz/tools) still go through the shared
  retrying transport. All ~25 call sites across CLI (`cmd.Context()`), TUI
  (`context.Background()` — no cancel plumbing yet, out of scope), and
  `pkg/accounts.ProbeType` updated. `usage.go`'s `runUsageWatch` loop now
  also selects on `ctx.Done()`. Verified: build/vet/test -race/gofmt/
  govulncheck all clean across the whole repo (root + api-explorer +
  playground submodules).
- [x] **C2 · Fix transport-bypassing services** (ex-B4, scope expanded) —
  `tools.go` was the known offender, but `account.go` and `quota.go` had the
  identical bug (private `http.Client{30s}` per call, no retry, no
  `parseAPIError`, hardcoded absolute URLs) — found during C1's call-site
  audit and fixed the same way. Added `ToolsBaseURL` const; all three now
  route through `doRequestBase`. Endpoints unchanged (not yet verified
  live — still C6).
- [x] **C3 · Multimodal messages** (ex-B1) — `Message` gained an `Images
  []string` field (`pkg/client/content.go`); `Content` itself stays a plain
  `string` (zero source-compat break for any existing code reading/writing
  it directly — the TUI's `renderMarkdown(msg.Content)` etc. needed no
  changes). Custom `MarshalJSON`/`UnmarshalJSON` switch the wire shape:
  plain `"content":"..."` when `Images` is empty (byte-identical to before),
  a content-parts array (`text` + `image_url` parts) when it isn't. CLI:
  `chat create --image <url>|@path` (repeatable), attaching to the last
  message; local files are base64-encoded as `data:` URIs with MIME
  guessed from the extension (falls back to `image/jpeg`). 9 new tests:
  5 in `pkg/client/types_test.go` (marshal/unmarshal both shapes,
  round-trip) + 4 in the new `chat_test.go` (main package's first test
  file) for `resolveImageArg`. TUI image attachment stays out of scope
  (Sprint D).
- [x] **C4 · Media-service tests + poll helper** — 15 new httptest cases
  across `images_test.go`/`videos_test.go`/`layout_test.go`/`audio_test.go`/
  `async_test.go`: wire format (multipart fields, default model, async path
  vs sync path), validation errors, APIError propagation through the shared
  transport, and the async lifecycle (PROCESSING→SUCCESS/FAIL). Added
  `Client.WaitForResult(ctx, id, interval)` — polls `GetAsyncResult` until a
  terminal state or context cancellation; the TUI still hand-rolls its own
  `tea.Tick` loop (needed for interim "polling…" UI state, which a blocking
  helper can't provide) but CLI/library callers can now use the blocking
  helper instead of hand-rolling retries. **Found in passing**: `contains()`
  in `models.go` is a prefix-check, not a substring-check — flagged
  separately as a follow-up task, not fixed here to keep this diff scoped to
  test coverage.
- [x] **C5 · Streaming timeout** (ex-Sprint-A follow-up) — root cause was
  `http.Client{Timeout: config.Timeout}`: Go's `Client.Timeout` bounds the
  *entire* request/response cycle including reading the body, so a
  long-running SSE stream got killed after `Config.Timeout` (default 30s)
  even mid-generation. Fixed by moving the timeout to the transport level
  (`DialContext`/`TLSHandshakeTimeout`/`ResponseHeaderTimeout`) instead of
  the whole-client timeout — this bounds "can we reach the server and start
  getting a response" without capping how long a live stream can keep
  producing tokens. No API change, no `Timeout: 0` workaround needed.
  Regression test: `TestCreateStreamSurvivesPastConfigTimeout` (a 90ms
  stream with a 10ms `Config.Timeout` completes all 3 chunks cleanly).
- [ ] **C6 · Live media smoke test** (user-gated: spends quota) — one cheap
  real call per new endpoint (`image/video/audio/ocr`) to confirm response
  shapes vs docs.

## Known limitations / follow-ups (from Sprint A)

- [ ] **Streaming timeout** — `Config.Timeout` (default 30s) is the http.Client's
  total deadline, which includes reading a streamed body. Long generations could be
  cut off. → tracked as **C5**.
- [ ] **CLI tool auto-execution** — `chat create --tool` declares tools and prints
  the model's `tool_calls` but does not execute them. An auto-exec loop needs a
  tool→command convention (security-gated). SDK `RunWithTools` covers the programmatic case.
- [ ] **Context plumbing** — `doRequest` (non-stream) uses `context.Background()`;
  only the streaming/tool-loop paths thread a caller context. The Phase 3 media
  services inherited the same gap. → tracked as **C1**.
- [ ] README architecture tree referenced a `cmd/` dir that doesn't exist — fixed
  in this pass; the repo is flat at root + `pkg/`.
- [x] `client.go` defines `AnthropicBaseURL` (`/api/anthropic`) — now used
  conceptually by `pkg/coding` (its own `AnthropicBaseURL(plan)` helper drives
  Claude Code / Factory Droid config). The `client` package const itself is still
  unused; keep as documentation or wire into an Anthropic-protocol client later.

## Sprint D candidates — gaps vs. the official Z.AI SDKs (research 2026-07-10)

Researched `docs.z.ai` (via its `llms.txt` index) and the official
`zai-org/z-ai-sdk-python` (the actively-maintained SDK; supersedes the
legacy `zhipuai`/`MetaGLM/zhipuai-sdk-python-v4` packages) to find capability
gaps between `zai-client` and what Z.AI actually ships. There is **no
official Z.AI CLI** — the only "zai-cli" on npm/GitHub
(`numman-ali/zai-cli`) is a third-party MCP-native tool (vision analysis,
web search/reader, GitHub repo exploration), not an official product, so it
isn't a parity target. The official SDK organizes its API surface into
`api_resource/{chat, images, videos, audio, tools, web_search, web_reader,
ocr, agents, assistant, batch, embeddings, files, file_parser, moderations,
voice}` — everything below is a real module in that list not yet in
`zai-client`, each verified against source (not just the README).

- [ ] **Already covered, no action needed** — confirmed while researching:
  Context Caching is fully automatic server-side (no request field to set);
  `Usage.PromptTokensDetails.CachedTokens` (`types.go`) already surfaces the
  `cached_tokens` field the docs describe. Tool-calling-in-stream ("Tool
  Streaming Output") is already handled via `StreamDelta.ToolCalls`.
- [ ] **Agents API** (`api_resource/agents`) — a generic `agent chat`
  endpoint (`docs.z.ai/api-reference/agents/agent.md`) plus three named
  agents: GLM Slide/Poster, Translation, Video Effect Template. Was already
  backlogged as one line; now confirmed as a real, documented resource
  distinct from regular chat completions (takes an agent/assistant ID, not
  just a model name).
- [ ] **Assistant API** (`api_resource/assistant`) — a *separate* resource
  from Agents (unclear from docs alone whether Agents supersedes/deprecates
  this, or they're parallel product lines — verify live before implementing
  either). Methods: `conversation()` (assistant_id + messages + optional
  streaming/attachments/metadata — supports tool types `code_interpreter`,
  `drawing_tool`, `function`, `retrieval`, `web_browser`), `query_support()`
  (list available assistants), `query_conversation_usage()` (paginated usage
  stats per assistant).
- [x] **Batch API** (`api_resource/batch`) — implemented in
  `pkg/client/batch.go`: `Create/Retrieve/List/Cancel` against
  `POST /batches`, `GET /batches/{id}`, `GET /batches` (cursor-paginated
  `after`/`limit`), `POST /batches/{id}/cancel`; `Batch.IsTerminal()` helper
  for the validating→in_progress→finalizing→completed/failed/expired/
  cancelled lifecycle. CLI: `batch create/status/list/cancel`. 6 tests in
  `pkg/client/batch_test.go`. Depends on Files (below) for `input_file_id`.
  Endpoints not yet verified live (same caveat as all Phase 3 media
  endpoints — C6-style follow-up).
- [x] **Files API** (`api_resource/files`) — implemented in
  `pkg/client/files.go`: `Upload/List/Delete/Content` against `POST /files`
  (multipart, not retried — re-uploading on transient failure is the
  caller's call, matching `AudioService.Transcribe`'s precedent),
  `GET /files`, `DELETE /files/{id}`, `GET /files/{id}/content`. CLI:
  `files upload/list/delete/download`. 5 tests in `pkg/client/files_test.go`.
  Endpoints not yet verified live.
- [ ] **Embeddings API** (`api_resource/embeddings`) —
  `/api/paas/v4/embeddings`. **Not implemented**: a live GitHub issue
  ([zai-org/z-ai-sdk-python#67](https://github.com/zai-org/z-ai-sdk-python/issues/67))
  reports "Unknown Model" errors calling embeddings on the global
  `api.z.ai` endpoint — may only work on the China `open.bigmodel.cn`
  endpoint today. Verify live on both endpoints before implementing;
  don't ship a service that silently only half-works.
- [ ] **Moderations API** (`api_resource/moderations`) — **not implemented**:
  unlike Batch/Files, the response shape (categories/scores/flagged fields)
  isn't documented anywhere — not in `docs.z.ai`'s doc index, not resolvable
  from the SDK's own type source (`moderation_completion.py` only shows a
  `model`/`input` echo, no results fields). Implementing this now would mean
  guessing JSON field names for something that fails silently (wrong names
  just deserialize to zero values, no error) rather than loudly. Needs a
  live call against the real API to capture an actual response body before
  writing the Go types.
- [ ] **Handwriting OCR** (`api_resource/ocr/handwriting_ocr.py`) — **not
  implemented**: request shape is known (`POST /files/ocr`,
  `tool_type: hand_write`, same multipart pattern as
  `pkg/client/layout.go`), but the response shape wasn't confirmed (unlike
  Files/Batch, couldn't pin it down from source in this pass) — same
  "verify before guessing types" concern as Moderations. Cheap to add once
  confirmed; distinct from the layout-parsing/glm-ocr endpoint already
  implemented (`/layout_parsing`).
- [ ] **Voice cloning API** (`api_resource/voice/voice.py`) — `/voice/clone`
  (audio sample + target text), `/voice/delete`, `/voice/list`. Not
  mentioned in the `docs.z.ai` `llms.txt` index at all (only
  `GLM-ASR-2512` transcription is documented there) — likely newer/beta or
  region-restricted. Verify it's live on the global endpoint before
  planning any work here.
- [ ] Anthropic-compatible endpoint wrapper (Z.AI ships `/api/anthropic`) —
  unchanged from prior backlog.
- [ ] Request/response logging + metrics collection — unchanged.
- [ ] Performance benchmarks — unchanged; no measured bottleneck exists yet,
  so this stays deferred until one does (per `golang-performance` guidance).
- [ ] **`GetAccountStatus`'s insufficient-balance branch is unreachable** —
  found while fixing the `contains()` prefix-bug (see Sprint C follow-ups
  above). `GetAccountStatus` calls `TestBalance` first, which already
  intercepts exactly the 1113/insufficient-balance case and returns its own
  clean `"insufficient balance: please recharge..."` message — by the time
  `GetAccountStatus` inspects `err.Error()`, the original "429"/"1113"
  markers its own classification looks for are already gone, so that
  specific `if` branch (usage.go, `strings.Contains(errMsg, "429") &&
  (...)`) can never match. Net effect: this specific case ends up with
  `APIAccessible=false` (arguably wrong — the API did respond) via the final
  `else`/`extractCleanError` fallback instead of the intended
  `APIAccessible=true, HasBalance=false` outcome. Locked in by
  `TestGetAccountStatusInsufficientBalanceViaTestBalanceShortcut` in
  `pkg/client/usage_test.go` (asserts current real behavior, not the
  intended one). Fix needs a design call: either have `TestBalance` stop
  transforming the message before `GetAccountStatus` classifies it, or have
  `GetAccountStatus` inspect the underlying `*APIError` (via `errors.As`)
  instead of string-matching a message it doesn't fully control.
