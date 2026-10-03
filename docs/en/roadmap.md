# Roadmap & Known Limitations

What's still open, why, and the exact action that closes it. Items leave this
page when a committed cassette or a merged change resolves them — this is the
verify-first convention's working list (see
[Contributing § the live-verification convention](../../CONTRIBUTING.md) and
[Architecture § the live-verification convention](architecture.md#the-live-verification-convention)
for the why).

The model catalog, API shapes, and Coding Plan behavior were last refreshed on
2026-10-02. `docs.z.ai` could not be fetched directly during that research, so
those facts come from search-result summaries of the official docs, the
official SDKs and npm package, and (where noted) third-party sources — which is
why several items below are listed as unverified rather than confirmed.

> **Recording a cassette yourself:** every item below that names a
> `TestVerify*` test has one that SKIPS until you capture a success-path
> cassette with `ZAI_RECORD=1`. The harness redacts `Authorization` to
> `Bearer REDACTED` before saving — confirm with
> `grep "Bearer " pkg/client/testdata/cassettes/<name>.yaml` before committing.
>
> ```sh
> ZAI_RECORD=1 ZAI_API_KEY=<real-key> go test -run TestVerify<Name> ./pkg/client
> ```

## Unverified live

Three groups: **services** whose success-path response shape isn't captured
yet, **chat-API fields** that match the docs but aren't pinned by a cassette,
and **data that is a snapshot** and needs periodic refresh.

### Services needing a success-path cassette

The dev account used so far has no PAYG balance / entitlement for these, so
only their request shape and error paths are confirmed. A cassette that
captures a real success response closes each item.

- **Account (biz) endpoints** — `AccountService.Balance`
  (`GET /api/biz/account/query-customer-account-report`) and
  `AccountService.Subscriptions` (`GET /api/biz/subscription/list`). Z.AI does
  not document the biz API; the routes and field names come from Z.AI's own
  ZCode client and community usage tools, and both are decoded through the
  shared `Envelope[T]`. NOT VERIFIED LIVE, on either region; record
  `TestVerifyAccountBalance` and `TestVerifyAccountSubscriptions` to settle it.
- **Embeddings, Moderations, and Rerank on `api.z.ai`**
  (`TestVerifyEmbeddings`, `TestVerifyModerations`) — it is unconfirmed
  whether these work on the international platform at all. Embeddings and
  Moderations are routed to `open.bigmodel.cn` (`BigModelBaseURL`); Rerank
  uses `Config.BaseURL`. Every account tested returned `400 Unknown Model`
  (code 1211) for the model IDs tried (embeddings and moderations on both
  hosts, rerank on `api.z.ai`); that looks like an entitlement gate, not a
  routing bug (see [Accounts & Quota](accounts-and-quota.md)), but the success
  shapes are unseen. There is no `TestVerify*` test for Rerank
  yet; `TestRerankCreateLive` only replays the 1211.
- **Anthropic Messages** (`TestVerifyAnthropicMessages`) — routing, the
  `anthropic-version` header, and Bearer auth are confirmed (a bogus key
  returns a clean 401, not a 404/timeout). Open question the cassette would
  settle: does GLM surface reasoning as Anthropic `thinking` blocks or the
  OpenAI-style `reasoning_content` field? `AnthropicResponse.Thinking()` reads
  both. ([claude-code-router#1133](https://github.com/musistudio/claude-code-router/issues/1133))
- **Agents `Invoke` success shape** (`TestVerifyAgentsInvoke`) — only the
  failure envelope (`ID`/`AgentID`/`Status`/`Error`) is live-confirmed today,
  via `testdata/cassettes/agents_invoke.yaml` (a 200-with-embedded-failure).
  The `Choices`/`Usage` success shape is modeled from docs only, as are the
  response shapes of `async-result` beyond its failure envelope
  (`agents_async_result.yaml`) and the content shape for image/video agents.
- **Voice `Clone` / `Delete`** (`TestVerifyVoiceClone`, `TestVerifyVoiceDelete`)
  — `Voice List` is confirmed live; clone/delete need an uploaded sample
  audio and a real cloned voice ID to record. Clone needs
  `ZAI_VOICE_SAMPLE_FILE_ID` + `ZAI_VOICE_NAME`; delete needs `ZAI_VOICE_ID`.
- **Quota and usage (monitor) endpoints** (`TestVerifyQuotaLimit`,
  `TestVerifyQuotaLimitChina`) — no committed cassette covers them yet. The
  credit-based plan shape (`CREDIT_LIMIT` windows, `QuotaTypeCreditLimit`)
  is modeled from third-party samples of the monitor API, and the
  `MonitorServerTZ` (UTC+8) assumption for the zoneless time strings was
  checked only against the global host.
- **Responses API** (`TestVerifyResponses`) — `ResponsesService` follows
  OpenAI's Responses spec; Z.AI documents the endpoint (for Codex) but not its
  schema. A recording would confirm the item and event shapes, which
  reasoning events GLM emits (`response.reasoning_text.delta` or
  `response.reasoning_summary_text.delta`), and whether pay-as-you-go keys
  are accepted or only Coding Plan keys.
- **Batch and Files endpoints generally** — no dedicated `TestVerify*`
  scaffold yet; would need an entitled PAYG account to record.

### Chat-API fields pending a cassette

These fields were added to `pkg/client/chat_types.go` / `chat.go` to match the
current docs.z.ai chat-completion spec and the official SDKs. They're
additive and unit-tested, but NOT VERIFIED LIVE until a cassette pins the exact
wire shape.

- **`ChatRequest.ToolStream`** (`TestVerifyChatStreamToolCall`) — GLM-4.6+
  streamed tool-call deltas. Cassette should show tool-call deltas arriving
  across multiple SSE chunks in `StreamDelta.ToolCalls`.
- **`Tool` discrimination across `function` / `retrieval` / `web_search` / `mcp`**
  (`NewFunctionTool` / `NewRetrievalTool` / `NewWebSearchTool` / `NewMCPTool`)
  — the spec lists all four types; only `function` is confirmed. The
  `web_search` payload shape follows the official docs and SDK examples, and
  the `mcp` tool and its `MCPCall` response entries are modeled from the docs
  only.
- **`ChatResponse.WebSearch`** (`TestVerifyChatWebSearchResponse`) — the
  top-level `web_search` array returned when a `web_search` tool fires.
  Entry shape reuses `WebSearchResult` from `tools.go` (live-verified for the
  standalone web-search tool); placement as a top-level array is modeled from
  the docs.
- **`ChatRequest.ReasoningEffort`** — validated locally against each model's
  catalog entry (`ReasoningEfforts`; GLM-5.3 family accepts `low`/`high`/`max`,
  GLM-5.2 every level). The per-model rules and the server's handling of an
  unsupported level come from the docs and SDKs; no cassette pins them.
- **Preserved thinking** — `ThinkingConfig.ClearThinking` and echoing
  `Message.ReasoningContent` back on later turns (which `RunWithTools` does
  automatically) follow the documented contract; the server's behavior when
  the reasoning is omitted or altered isn't captured.
- **Video and file input** — `Message.Videos` / `Message.Files` encode as
  `video_url` / `file_url` content parts for natively multimodal models
  (`DefaultVisionModel`); the encoding follows the docs and has no cassette.
- **`FinishReason*` constants** (`sensitive`, `model_context_window_exceeded`,
  `network_error`) — added from the docs; no cassette reproduces these
  termination paths yet.
- **Client-side tool-name regex** (`^[A-Za-z0-9_-]{1,64}$`) and **128-function
  cap** — documented server-side rules we enforce locally; not confirmed as
  the server's exact rejection criteria.
- **China regional gateway for monitor/biz/agents/Anthropic/detection** —
  `RegionChina` routes these to `open.bigmodel.cn`. `/models` and
  `/chat/completions` are live-verified on the China host; the other paths are
  modeled by mirroring `api.z.ai`'s layout (`Region` in `region.go`) and need a
  cassette against an entitled China key to confirm.

### Snapshot data that needs periodic refresh

- **Model catalog** (`pkg/client/models_catalog.go`) — IDs, context sizes,
  output caps, prices, capabilities, and reasoning efforts are a snapshot
  transcribed on 2026-10-02 from the docs.z.ai pricing and model pages (via
  search summaries; some rows rely on third-party sources, and the status of
  `glm-5-turbo` / `glm-5v-turbo` is uncertain). The API gives no signal when
  any of it changes. The header of the file lists the refresh steps.
- **Peak-hours and off-peak promotion** (`internal/usageview/peak.go`) — the
  weekday 14:00-18:00 UTC+8 peak window and the billing multipliers are
  mirrored client-side because the quota API doesn't expose them, and the
  `offPeakPromotions` entry currently covers a promotion that ends
  2026-10-07. Remove or update it when that passes.

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

## Follow-ups

- **Record the pending verification cassettes** — the `TestVerify*` tests
  above exist and skip until recorded. Run them with a real key (a
  pay-as-you-go one for the balance, a Coding Plan one for quota, Responses,
  and subscriptions, and a China key for `TestVerifyQuotaLimitChina`), e.g.
  `ZAI_RECORD=1 ZAI_API_KEY=… go test -run 'TestVerify(AccountBalance|AccountSubscriptions|QuotaLimit|Responses)$' ./pkg/client -v`,
  then fix any field names the real responses contradict. The recording hook
  redacts the key and account identifiers; check the YAML before committing.
- **Codex hosted MCP servers** — `go-z-ai coding mcp add codex` offers only
  the Vision server, as the official helper does, because of
  openai/codex#14793 (Codex rejecting a reply without a Content-Type, which
  Z.AI's hosted servers send to `notifications/initialized`). The upstream
  issue is now closed: once a current Codex release is confirmed to accept
  the hosted servers, give `codex` a remote MCP entry shape (`url` plus
  `http_headers`).
- **Agent conversation endpoint** — not implemented. `POST /v1/agents/conversation`
  returns the conversation history of an agent that runs conversations (docs
  say it supports only `slides_glm_agent`). `AgentAsyncResultRequest` already
  carries `ConversationID`; the history call itself is missing. Related:
  `AgentInvokeRequest.Stream` exists on the wire type, but there is no
  streaming decoder for agents, so streamed agent responses are unsupported.

## Not implemented

- **Performance benchmarks** — deferred until a real bottleneck is measured;
  no known hot path currently justifies one (profile before optimizing).

## Shipped (kept for orientation)

- **Request/response observability hooks** — landed in v0.2.0. The `Hook`
  seam in `pkg/client` (`OnRequest`/`OnResponse`/`OnError`/`OnStreamChunk`)
  fires on every request lifecycle event, paired one terminal call per
  attempt; the first concrete implementation is `OTelHook` in `pkg/observe`
  (OpenTelemetry spans + metrics). See
  [Library Guide → Observability hooks](library-guide.md#observability-hooks).

## Deliberately not implemented

- **Assistant API** — confirmed deprecated. Z.AI's own live OpenAPI spec
  (`docs.bigmodel.cn/openapi/openapi.json`) marks every Assistant path
  `"deprecated": true`, and calling it from `api.z.ai` times out entirely
  rather than erroring. Building a client for a sunset API isn't worth the
  maintenance surface — if Z.AI ever un-deprecates it, the spec above has
  the full request/response schemas ready to transcribe.
