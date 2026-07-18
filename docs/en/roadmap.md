# Roadmap & Known Limitations

What's still open, why, and the exact action that closes it. Items leave this
page when a committed cassette or a merged change resolves them — this is the
verify-first convention's working list (see
[Contributing § the live-verification convention](../../CONTRIBUTING.md) and
[Architecture § the live-verification convention](architecture.md#the-live-verification-convention)
for the why).

> **Recording a cassette yourself:** every "Unverified live" item below has a
> matching `TestVerify*` test that SKIPS until you capture a success-path
> cassette with `ZAI_RECORD=1`. The harness redacts `Authorization` to
> `Bearer REDACTED` before saving — confirm with
> `grep "Bearer " pkg/client/testdata/cassettes/<name>.yaml` before committing.
>
> ```sh
> ZAI_RECORD=1 ZAI_API_KEY=<real-key> go test -run TestVerify<Name> ./pkg/client
> ```

## Unverified live

Two groups: **services** whose success-path response shape isn't captured
yet, and **fields added in the 2026-07-18 sprint** that match the docs but
aren't pinned by a cassette.

### Services needing a success-path cassette

The dev account used so far has no PAYG balance / entitlement for these, so
only their request shape and error paths are confirmed. A cassette that
captures a real success response closes each item.

- **Anthropic Messages** (`TestVerifyAnthropicMessages`) — routing, the
  `anthropic-version` header, and Bearer auth are confirmed (a bogus key
  returns a clean 401, not a 404/timeout). Open question the cassette would
  settle: does GLM surface reasoning as Anthropic `thinking` blocks or the
  OpenAI-style `reasoning_content` field? ([claude-code-router#1133](https://github.com/musistudio/claude-code-router/issues/1133))
- **Embeddings** (`TestVerifyEmbeddings`) — currently returns `400 Unknown
  Model` (code 1211) on every account tested; that's an entitlement gate,
  not a routing bug (see [Accounts & Quota](accounts-and-quota.md)).
- **Moderations** (`TestVerifyModerations`) — same 1211 entitlement gate.
- **Agents `Invoke` success shape** (`TestVerifyAgentsInvoke`) — only the
  failure envelope (`ID`/`AgentID`/`Status`/`Error`) is live-confirmed today,
  via `testdata/cassettes/agents_invoke.yaml` (a 200-with-embedded-failure).
  The `Choices`/`Usage` success shape is modeled from docs only.
- **Voice `Clone` / `Delete`** (`TestVerifyVoiceClone`, `TestVerifyVoiceDelete`)
  — `Voice List` is confirmed live; clone/delete need an uploaded sample
  audio and a real cloned voice ID to record. Clone needs `ZAI_VOICE_SAMPLE_FILE_ID`
  + `ZAI_VOICE_NAME`; delete needs `ZAI_VOICE_ID`.
- **Batch and Files endpoints generally** — no dedicated `TestVerify*`
  scaffold yet; would need an entitled PAYG account to record.

### Fields added in the 2026-07-18 sprint, pending a cassette

These fields were added to `pkg/client/types.go` / `chat.go` to match the
current docs.z.ai chat-completion spec. They're additive and unit-tested,
but NOT VERIFIED LIVE until a cassette pins the exact wire shape. Each has a
`TestVerify*` test ready to record.

- **`ChatRequest.StreamToolCall`** (`TestVerifyChatStreamToolCall`) — GLM-4.6+
  streamed tool-call deltas. Cassette should show tool-call deltas arriving
  across multiple SSE chunks in `StreamDelta.ToolCalls`.
- **`Tool` discrimination across `function` / `retrieval` / `web_search`**
  (`NewFunctionTool` / `NewRetrievalTool` / `NewWebSearchTool`) — the spec
  lists all three types; only `function` is confirmed. The `web_search`
  payload shape (`{"search_query":[...]}`) follows the official Python SDK
  example.
- **`ChatResponse.WebSearch`** (`TestVerifyChatWebSearchResponse`) — the
  top-level `web_search` array returned when a `web_search` tool fires.
  Entry shape reuses `WebSearchResult` from `tools.go` (live-verified for the
  standalone web-search tool); placement as a top-level array is modeled from
  the docs.
- **`ThinkingConfig.Effort = "xhigh"`** — added to the validated enum
  (`xhigh`→`max`, GLM-5.2 only). No dedicated test; covered by any thinking
  + xhigh cassette.
- **`FinishReason*` constants** (`sensitive`, `model_context_window_exceeded`,
  `network_error`) — added from the docs; no cassette reproduces these
  termination paths yet.
- **Client-side tool-name regex** (`^[A-Za-z0-9_-]{1,64}$`) and **128-function
  cap** — documented server-side rules we enforce locally; not confirmed as
  the server's exact rejection criteria.
- **China regional gateway for monitor/biz/agents/detection** — `RegionChina`
  routes quota/usage/account/agents/detection to `open.bigmodel.cn`.
  `/models` and `/chat/completions` are live-verified on the China host; the
  monitor/biz/agents paths are modeled by mirroring `api.z.ai`'s layout and
  need a cassette against an entitled China key to confirm.

### Older open questions (no dedicated test yet)

- **Tool-schema compatibility rewriting** — the set of JSON-Schema constructs
  GLM's parser rejects with HTTP 500 (`anyOf`/`oneOf`/`allOf`/`$ref`) is
  drawn from community bug reports
  ([claude-code-router#1474](https://github.com/musistudio/claude-code-router/issues/1474)),
  not reproduced against a live account here. The rewrite itself is fully
  unit-tested and inert on already-flat schemas; a cassette pinning exactly
  which constructs 500 (and which the flattened output makes pass) would
  upgrade this from "documented behavior" to "live-verified." See
  `pkg/client/toolschema.go`.

## Not implemented

- **Request/response logging and metrics collection** — no built-in
  instrumentation hooks yet.
- **Performance benchmarks** — deferred until a real bottleneck is measured;
  no known hot path currently justifies one (profile before optimizing).

## Deliberately not implemented

- **Assistant API** — confirmed deprecated. Z.AI's own live OpenAPI spec
  (`docs.bigmodel.cn/openapi/openapi.json`) marks every Assistant path
  `"deprecated": true`, and calling it from `api.z.ai` times out entirely
  rather than erroring. Building a client for a sunset API isn't worth the
  maintenance surface — if Z.AI ever un-deprecates it, the spec above has
  the full request/response schemas ready to transcribe.
