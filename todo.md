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

## Sprint B — Generation, multimodal, correctness

- [ ] **B1 · Multimodal messages** — `Message.Content` → `string | []ContentPart`
  (backward-compatible `MarshalJSON`); CLI `--image url|@path`.
- [ ] **B2 · Image generation service** — `pkg/client/images.go`, CogView-4 / GLM-Image
  (verify endpoint against live docs); CLI `images generate`.
- [ ] **B3 · Video generation service** — `pkg/client/videos.go`, async submit+poll
  (CogVideoX-3, Vidu); CLI `videos create/get`. Verify endpoints.
- [ ] **B4 · Fix `pkg/client/tools.go`** — route through `httpClient` + `parseAPIError`
  (currently own client per method, hardcoded/speculative endpoints); reconcile with
  `quota.go` tool codes (`search-prime`/`web-reader`/`zread`).
- [ ] **B5 · Tests + catalog hardening** — golden tests for B1–B3; drive model catalog
  from live `/models`; reconcile GLM-5.x reality vs the `docs.z.ai` nav (still 4.6).

## Known limitations / follow-ups (from Sprint A)

- [ ] **Streaming timeout** — `Config.Timeout` (default 30s) is the http.Client's
  total deadline, which includes reading a streamed body. Long generations could be
  cut off. Follow-up: per-request timeout via context (the SDK already accepts a
  `ctx`), and document setting `Timeout: 0` for long streams.
- [ ] **CLI tool auto-execution** — `chat create --tool` declares tools and prints
  the model's `tool_calls` but does not execute them. An auto-exec loop needs a
  tool→command convention (security-gated). SDK `RunWithTools` covers the programmatic case.
- [ ] **Context plumbing** — `doRequest` (non-stream) uses `context.Background()`;
  only the streaming/tool-loop paths thread a caller context. Acceptable for the CLI
  (Ctrl+C kills the process) but worth threading through for library callers.
- [ ] README architecture tree referenced a `cmd/` dir that doesn't exist — fixed
  in this pass; the repo is flat at root + `pkg/`.
- [x] `client.go` defines `AnthropicBaseURL` (`/api/anthropic`) — now used
  conceptually by `pkg/coding` (its own `AnthropicBaseURL(plan)` helper drives
  Claude Code / Factory Droid config). The `client` package const itself is still
  unused; keep as documentation or wire into an Anthropic-protocol client later.

## Backlog (deferred)

- [ ] Agents: GLM Slide/Poster, Translation, Video Effect Template.
- [ ] Batch request handling.
- [ ] Anthropic-compatible endpoint wrapper (Z.AI ships `/api/anthropic`).
- [ ] Request/response logging + metrics collection.
- [ ] Performance benchmarks.
