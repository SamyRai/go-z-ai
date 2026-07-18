# Roadmap & Known Limitations

## Unverified live

A few services are implemented from Z.AI's documented OpenAPI spec but their
*success* response shape hasn't been confirmed against a real successful
call (only the request shape and error paths have) — the account used for
development has no PAYG balance/entitlement for these.

**Verifying these yourself:** `pkg/client/live_verify_test.go` holds a
verification test per service that replays a committed cassette offline (and
skips until one exists). If you have an entitled account, record one with:

```sh
ZAI_RECORD=1 ZAI_API_KEY=<real-key> go test -run TestVerifyAnthropicMessages ./pkg/client
```

The API key is redacted out of the cassette at save time; confirm with
`grep "Bearer " pkg/client/testdata/cassettes/<name>.yaml` (must read
`Bearer REDACTED`) before committing. If you hit a shape mismatch instead,
please [open an issue](https://github.com/SamyRai/go-z-ai/issues) or PR the
cassette. Still unverified:

- Agents `Invoke`'s success-path response shape (`Choices`/`Usage`)
- Embeddings and Moderations' actual output (entitlement-gated on every
  account tested so far — see [Accounts & Quota](accounts-and-quota.md))
- Voice `Clone`/`Delete` (`List` is confirmed live and working)
- Batch and Files endpoints generally
- **Anthropic Messages** (`AnthropicService`, `POST /api/anthropic/v1/messages`)
  — routing, the `anthropic-version` header, and Bearer auth are confirmed
  reaching the live endpoint (a bogus key returns a clean HTTP 401, not a
  404/timeout), but the *success*-path response body (`content` blocks,
  `stop_reason`, `usage`) is modeled from Anthropic's documented shape and not
  yet parsed from a real entitled call here. In particular, whether GLM returns
  reasoning as Anthropic `thinking` blocks or in an OpenAI-style
  `reasoning_content` field (the claude-code-router#1133 case) is unconfirmed —
  `AnthropicResponse.Thinking()` reads both, but which one the endpoint actually
  populates needs a live capture.
- **Tool-schema compatibility rewriting** — the set of JSON-Schema constructs
  GLM's parser rejects with HTTP 500 (`anyOf`/`oneOf`/`allOf`/`$ref`) is drawn
  from community bug reports (e.g.
  [claude-code-router#1474](https://github.com/musistudio/claude-code-router/issues/1474)),
  not yet reproduced against a live account here. The rewrite itself is
  fully unit-tested and inert on already-flat schemas; if you can record a
  live cassette that pins down exactly which constructs 500 (and which the
  flattened output makes pass), that would upgrade this from "documented
  behavior" to "live-verified." See `pkg/client/toolschema.go`.
- **Chat-completion API additions** — the following fields were added to
  `pkg/client/types.go` to match the current docs.z.ai chat-completion spec,
  and are `NOT VERIFIED LIVE` until a cassette pins them:
  - `ChatRequest.StreamToolCall` (GLM-4.6+ streamed tool-call deltas).
  - `Tool` discrimination across `function` / `retrieval` / `web_search`
    types (the spec lists three types but notes only `function` is fully
    supported; the retrieval/web_search request shapes are modeled, not
    confirmed). See `NewRetrievalTool` / `NewWebSearchTool`.
  - `ChatResponse.WebSearch` (top-level `web_search` array; the entry shape
    reuses `WebSearchResult` from `tools.go`, which is live-verified for the
    standalone web-search tool).
  - `ThinkingConfig.Effort` now documents `xhigh` (GLM-5.2; `xhigh`→`max`).
  - `FinishReason*` constants for the live values `sensitive`,
    `model_context_window_exceeded`, `network_error`.
  - The client-side tool-name regex guard (`^[A-Za-z0-9_-]{1,64}$`) and the
    128-function cap — these are documented rules, not yet confirmed as the
    server's exact rejection criteria.
- **China regional gateway for monitor/biz/agents** — `Config.Region =
  RegionChina` routes the quota/usage, account, and agents calls to
  `open.bigmodel.cn`. The China mirror hosts are modeled by mirroring the
  `api.z.ai` path layout (live-verified for `/models` and `/chat/completions`,
  see `BigModelBaseURL`); the monitor/biz/agents hosts on the China side are
  `NOT VERIFIED LIVE`. Pin with `ZAI_RECORD=1` against an entitled China key.

## Not implemented

- **Request/response logging and metrics collection** — no built-in
  instrumentation hooks yet.
- **Performance benchmarks** — deferred until a real bottleneck is measured;
  no known hot path currently justifies one (see the `golang-performance`
  guidance this repo follows: profile before optimizing).

## Deliberately not implemented

- **Assistant API** — confirmed deprecated. Z.AI's own live OpenAPI spec
  (`docs.bigmodel.cn/openapi/openapi.json`) marks every Assistant path
  `"deprecated": true`, and calling it from `api.z.ai` times out entirely
  rather than erroring. Building a client for a sunset API isn't worth the
  maintenance surface — if Z.AI ever un-deprecates it, the spec above has
  the full request/response schemas ready to transcribe.
