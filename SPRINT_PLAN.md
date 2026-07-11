# Sprint Plan — Better Z.AI API support in `zai-client`

> Planned 2026-07-08. **Sprint A delivered 2026-07-08.** Operational checklist:
> [`todo.md`](./todo.md). API facts reconciled against the live
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

6. **B1 · Multimodal messages (M)** — backward-compatible `Content` type, `--image`.
7. **B2 · Image generation service (M)** — `images.go`, CogView-4 / GLM-Image.
8. **B3 · Video generation service (M)** — `videos.go`, async submit + poll.
9. **B4 · Fix `tools.go` (S)** — route through `httpClient`, verify endpoints.
10. **B5 · Tests + catalog hardening (M)** — golden tests; drive `/models` live.

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
