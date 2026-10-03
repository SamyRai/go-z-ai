# Library Guide

`pkg/client` is a standalone Go library (standard library only) — everything
the CLI does, it does by calling this package. You can depend on it directly
without the CLI at all.

## Contents

- [Creating a client](#creating-a-client)
- [Services](#services)
- [Default models](#default-models)
- [Chat completions](#chat-completions)
- [Anthropic-compatible Messages API](#anthropic-compatible-messages-api)
- [Responses API (Codex protocol)](#responses-api-codex-protocol)
- [Account detection and status](#account-detection-and-status)
- [Error handling](#error-handling)
- [Observability hooks](#observability-hooks)
- [Multi-account credential management](#multi-account-credential-management)
- [Testing your own code against this client](#testing-your-own-code-against-this-client)
- [Architecture notes](#architecture-notes)

```bash
go get github.com/SamyRai/go-z-ai
```

The module requires Go 1.26 (see `go.mod`).

```go
import "github.com/SamyRai/go-z-ai/pkg/client"
```

## Creating a client

```go
c, err := client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
})
if err != nil {
    log.Fatal(err)
}
```

Or, to configure everything from the environment — the same variables the CLI
honors:

```go
c, err := client.NewClientFromEnv()
if err != nil {
    log.Fatal(err)
}
```

| Variable | Maps to | Notes |
|---|---|---|
| `ZAI_API_KEY` | `Config.APIKey` | Required |
| `ZAI_API_BASE_URL` | `Config.BaseURL` | Optional chat/PaaS root override |
| `ZAI_REGION` | `Config.Region` | Parsed with `client.ParseRegion`: `global` (default) or `china` |
| `ZAI_CHINA_API_KEY` | `Config.ChinaAPIKey` | Optional separate bigmodel.cn credential |
| `ZAI_MONITOR_TIMEZONE` | `Config.MonitorTimezone` | IANA name, `UTC`, offset such as `UTC+8`, or `local`; a bad value is an error |

`Config` fields:

| Field | Default | Notes |
|---|---|---|
| `APIKey` | — | Required |
| `BaseURL` | `Region.PaaSBaseURL()` — `https://api.z.ai/api/paas/v4` for the global region | An explicit value always wins. Coding Plan keys use `Region.CodingBaseURL()` (or `DetectedAccount.BaseURL`, see [Account detection and status](#account-detection-and-status)). |
| `HTTPClient` | an internally configured `*http.Client` | Bring your own transport if you need custom TLS/proxy behavior |
| `Timeout` | 30s (`DefaultTimeout`) | Bounds dial/TLS/response-header wait — **not** the whole response body read, so it never truncates a live SSE stream |
| `MaxRetries` | 3 | Retries on 429/5xx/network errors. `-1` disables retries entirely |
| `RetryDelay` | 200ms | Base exponential-backoff delay |
| `ChinaAPIKey` | falls back to `APIKey` | Only needed if you hold a separate bigmodel.cn-only credential — see [Accounts & Quota](accounts-and-quota.md#regional-gateways-apizai--openbigmodelcn) |
| `Region` | `RegionGlobal` | Selects the gateway: `RegionGlobal` (api.z.ai) or `RegionChina` (open.bigmodel.cn). See [Regions](#regions). |
| `MonitorTimezone` | `client.MonitorServerTZ` (UTC+8) | Timezone the monitor (quota/usage) API's zoneless timestamps are interpreted in. Override only if a live capture shows a different server zone. |
| `DisableToolSchemaCompat` | `false` | Send tool schemas untouched — see [Tool-schema compatibility](#tool-schema-compatibility) |
| `UserAgent` | `"go-z-ai/<version>"` | Overrides the `User-Agent` header sent on every request. The default identifies go-z-ai to Z.AI's API — important under the coding endpoint's [usage policy](coding-tools.md#compliance--usage-policy-). Override only when you need a distinct identifier (a downstream app, a proxy, an MCP server); the override string is sent verbatim. |
| `Hooks` | `nil` | Observability hooks ([`Hook`](library-guide.md#observability-hooks)) that fire on every request/response/error/stream-chunk. Empty by default — the no-hook path is zero-cost. Concrete implementations live in `pkg/observe` (OpenTelemetry). |

Every service method takes `context.Context` as its first argument and
propagates it all the way to the HTTP call — cancel it to abort a request or
a pending retry backoff.

### Regions

`Region` owns every gateway URL; nothing else in the library hard-codes a
host. `Config.Region` picks the host for the region-scoped services — monitor
(quota/usage), biz (account), agents, the Anthropic-compatible Messages API,
the Responses API, and account-type detection — and, when `BaseURL` is empty, the default chat
root. `c.Region()` returns the configured value; an empty or unrecognized
`Region` means global.

| Method | Path under `Region.Host()` |
|---|---|
| `PaaSBaseURL()` | `/api/paas/v4` — pay-as-you-go, OpenAI-compatible |
| `CodingBaseURL()` | `/api/coding/paas/v4` — GLM Coding Plan |
| `AnthropicBaseURL()` | `/api/anthropic` |
| `ResponsesBaseURL()` | `/api/v1` — OpenAI Responses protocol (what Codex uses) |
| `MonitorBaseURL()` | `/api/monitor` — quota/usage |
| `BizBaseURL()` | `/api/biz` — account |
| `AgentsBaseURL()` | `/api` |
| `MCPServerURL(name)` | `/api/mcp/<name>/mcp` — Z.AI's hosted MCP servers |

`Host()` is `https://api.z.ai` or `https://open.bigmodel.cn` (`client.GlobalHost`
/ `client.ChinaHost`); `ConsoleURL()` is the web console
(`https://z.ai` / `https://bigmodel.cn`); `BaseURLFor(accountType)` returns the
coding or PaaS chat root for an `AccountType`. `client.ParseRegion` accepts
`china` (aliases `cn`, `bigmodel`) and `global` (aliases `west`, empty),
case-insensitively; an unknown value resolves to global rather than failing.

```go
c, err := client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
    Region: client.RegionChina, // chat, quota, account, agents, Anthropic -> open.bigmodel.cn
})
if err != nil {
    log.Fatal(err)
}
```

Embeddings and Moderations are the exception: they always call
`client.BigModelBaseURL` (open.bigmodel.cn) with `ChinaAPIKey`, whatever the
`Region`. The China mirror of the monitor/biz/agents routes is NOT VERIFIED
LIVE — see [Roadmap](roadmap.md).

## Services

`Client` exposes one method per service, all following the same
`c.<Service>().<Method>(ctx, ...)` shape:

| Accessor | Covers |
|---|---|
| `c.Chat()` | Completions — `Create`, `CreateAsync`, `Stream`, `RunWithTools`, `RunWithToolsLimit` |
| `c.Models()` | `List`, `Get`, `GetTextModels`, `GetVisionModels`, `GetFreeModels`, `RefreshCache` |
| `c.Images()` | `Generate`, `GenerateAsync` |
| `c.Videos()` | `Generate` (always async) |
| `c.Audio()` | `Transcribe`, `Speech` |
| `c.Voice()` | `Clone`, `Delete`, `List` — GLM-TTS voice cloning |
| `c.Layout()` | `Parse`, `HandwritingOCR` |
| `c.FileParser()` | `Create`, `Sync`, `Result` — document-to-text for RAG |
| `c.Files()` | `Upload`, `List`, `Delete`, `Content` |
| `c.Batch()` | `Create`, `Retrieve`, `List`, `Cancel` |
| `c.Agents()` | `Invoke`, `AsyncResult` |
| `c.Anthropic()` | `Create`, `Stream` — Anthropic-protocol `/v1/messages` surface |
| `c.Responses()` | `Create`, `Stream` — OpenAI Responses-protocol `/api/v1/responses`, the Codex surface |
| `c.Embeddings()` | `Create` (routes to `open.bigmodel.cn`) |
| `c.Moderations()` | `Create` (routes to `open.bigmodel.cn`) |
| `c.Rerank()` | `Create` |
| `c.Tools()` | `WebSearch`, `WebReader`, `Tokenize` |
| `c.Quota()` | GLM Coding Plan quota and usage — `GetQuotaLimit`, `GetModelUsage`, `GetToolUsage` |
| `c.Detection()` | `DetectAccountType`, `CheckAccountStatus` |
| `c.Account()` | `Balance`, `Subscriptions` |
| `c.GetAsyncResult(ctx, id)`, `c.WaitForResult(ctx, id, interval)` | Shared polling for async image/video/chat tasks |

Every request-validation check (required fields, etc.) happens client-side
before a request is sent — you get a local `error` immediately rather than a
round trip for something like a missing `model`.

There is no client-side usage tracker: compute a request's cost with
`Pricing.Cost` (see [Response fields worth
checking](#response-fields-worth-checking)) and read plan consumption from
`c.Quota()`.

## Default models

Use these constants instead of hard-coding model IDs; they are the same
defaults the CLI, the TUI, and the coding-tool config writers use. The
catalog behind them (`pkg/client/models_catalog.go`) also carries context
size, output cap, prices, capabilities, and accepted reasoning efforts for
every model; `client.CatalogEntry(id)` returns one row and `c.Models().List`
merges the catalog into the live `/models` listing.

| Constant | Model | Use |
|---|---|---|
| `DefaultModel` | `glm-5.3` | Flagship text model for reasoning, coding, and agentic work (1M context) |
| `DefaultFastModel` | `glm-5.3-flash` | Fast, low-cost tier; natively multimodal |
| `DefaultVisionModel` | `glm-5.3-flash` | Same as `DefaultFastModel`: image, video, and file input |
| `DefaultOCRModel` | `glm-ocr` | `Layout().Parse` |
| `DefaultASRModel` | `glm-asr-2512` | `Audio().Transcribe` |
| `DefaultTTSModel` | `glm-tts` | `Audio().Speech` |

Image models are `client.ModelGLMImage` (the default; the only one that
supports `GenerateAsync`) and `client.ModelCogView4`. Video models are in
`client.VideoModels`, led by `client.ModelCogVideoX3`.

## Chat completions

```go
resp, err := c.Chat().Create(ctx, client.ChatRequest{
    Model: client.DefaultModel,
    Messages: []client.Message{
        {Role: "system", Content: "You are a helpful assistant."},
        {Role: "user", Content: "Explain goroutines in one paragraph"},
    },
    Temperature: 0.7,
})
if err != nil {
    return err
}
fmt.Println(resp.Choices[0].Message.Content)
```

`Temperature`, `TopP`, and `MaxTokens` default to the server's values when
zero; `Temperature` and `TopP` must be within [0, 1]. `DoSample` is a
`*bool` so an explicit `false` is sent. `RequestID` and `UserID` (6–128
characters, for abuse monitoring) are optional.

### Streaming

`Stream` returns an iterator (`iter.Seq2[StreamChunk, error]`) you range over;
it sets `stream=true` for you:

```go
for chunk, err := range c.Chat().Stream(ctx, req) {
    if err != nil {
        return err // terminal — the loop ends after this iteration
    }
    if len(chunk.Choices) > 0 {
        fmt.Print(chunk.Choices[0].Delta.Content)
    }
}
```

The stream runs synchronously inside the loop — there is no producer
goroutine. Breaking out of the loop stops reading and closes the connection,
and cancelling `ctx` aborts a blocked read. Connect-phase transient failures
(429/5xx/network) are retried up to `Config.MaxRetries` exactly like `Create`;
once a stream has begun, a failure surfaces as the terminal `err` and is never
retried, because part of the response was already consumed. A request that
fails validation yields one error from the iterator before anything is sent.

Reasoning arrives in `chunk.Choices[0].Delta.ReasoningContent`, and the final
chunk carries `chunk.Usage`.

#### In-band stream errors

The server can report a failure after the HTTP 200 has been sent: an
OpenAI-style `{"error": {...}}` chunk, an Anthropic `event: error`, or a
Responses `error` / `response.failed` event. The client turns each into an
`*APIError` (with `HTTPStatus` 200) and ends the
stream with it as the terminal `err`, so the same `errors.As` handling works
for both kinds of failure:

```go
for chunk, err := range c.Chat().Stream(ctx, req) {
    if err != nil {
        var apiErr *client.APIError
        if errors.As(err, &apiErr) {
            log.Printf("stream failed: [%d] %s", apiErr.Code, apiErr.UserMessage)
        }
        return err
    }
    _ = chunk
}
```

An in-band error is not retried even when its code is marked retriable: the
stream had already started. See [Error Handling](error-handling.md).

#### Streaming tool calls

Set `req.ToolStream = true` (GLM-4.6 and later) to receive tool-call
arguments incrementally in `chunk.Choices[0].Delta.ToolCalls` across multiple
events, rather than as one batch at the end of the turn. Fragments of the same
call share a `ToolCall.Index`; concatenate their `Function.Arguments`. Useful
for surfacing "the model is calling a tool…" progress to a UI. NOT VERIFIED
LIVE — see [Roadmap](roadmap.md).

```go
req.ToolStream = true
calls := map[int]*client.FunctionCall{}
for chunk, err := range c.Chat().Stream(ctx, req) {
    if err != nil {
        return err
    }
    for _, choice := range chunk.Choices {
        for _, tc := range choice.Delta.ToolCalls {
            if tc.Function == nil {
                continue
            }
            fc := calls[tc.Index]
            if fc == nil {
                fc = &client.FunctionCall{}
                calls[tc.Index] = fc
            }
            if tc.Function.Name != "" {
                fc.Name = tc.Function.Name
            }
            fc.Arguments += tc.Function.Arguments
        }
    }
}
```

### Reasoning effort

`ChatRequest.ReasoningEffort` sets how hard a thinking model reasons. Use the
`Effort*` constants (`EffortMax`, `EffortXhigh`, `EffortHigh`, `EffortMedium`,
`EffortLow`, `EffortMinimal`, `EffortNone`); `client.AllEfforts` lists them
all. The API default is `max`.

```go
resp, err := c.Chat().Create(ctx, client.ChatRequest{
    Model:           client.DefaultModel,
    Messages:        []client.Message{{Role: "user", Content: "Plan a zero-downtime schema migration"}},
    ReasoningEffort: client.EffortHigh,
})
if err != nil {
    return err
}
msg := resp.Choices[0].Message
fmt.Println(msg.ReasoningContent) // the reasoning
fmt.Println(msg.Content)          // the answer
if d := resp.Usage.CompletionTokensDetails; d != nil {
    fmt.Println("reasoning tokens:", d.ReasoningTokens)
}
```

The value is validated client-side against the model's catalog entry
(`ModelCatalogEntry.ReasoningEfforts`, also exposed as
`ModelDetails.ReasoningEfforts`), so an unsupported level is a local error
instead of a server round trip:

- `glm-5.2` accepts every level.
- The GLM-5.3 family (`glm-5.3`, `glm-5.3-flash`, `glm-5.3-flashx`) accepts
  only `low`, `high`, and `max`, and always thinks — disabling thinking is not
  supported there.
- A cataloged model with no effort list (for example `glm-5.1`) rejects the
  parameter.
- A model that is not in the catalog is checked only against `AllEfforts`.

The server normalizes `none`/`minimal` to "skip thinking", `low`/`medium` to
`high`, and `xhigh` to `max`. The levels follow docs.z.ai; they are not
live-verified here.

To keep reasoning across turns (preserved thinking), set
`ThinkingConfig.ClearThinking` to false and echo each assistant turn's
`ReasoningContent` back unchanged. `RunWithTools` does this for you.

```go
keep := false
req.Thinking = &client.ThinkingConfig{Type: client.ThinkingEnabled, ClearThinking: &keep}
req.Messages = append(req.Messages, client.Message{
    Role:             "assistant",
    Content:          msg.Content,
    ReasoningContent: msg.ReasoningContent,
})
```

### Async

```go
task, err := c.Chat().CreateAsync(ctx, req)
if err != nil {
    return err
}
result, err := c.WaitForResult(ctx, task.ID, 3*time.Second)
if err != nil {
    return err
}
if result.TaskStatus == client.TaskStatusSuccess {
    fmt.Println(result.Choices[0].Message.Content)
}
```

`WaitForResult` returns when the task leaves `PROCESSING`, successfully or
not, so check `TaskStatus` (`TaskStatusSuccess` / `TaskStatusFail`). Image and
video tasks use the same polling. An image task's results are in
`AsyncResultResponse.ImageResult` (each a `GeneratedImage` with a `URL` that
expires after 30 days); a video task's are in `VideoResult`:

```go
task, err := c.Images().GenerateAsync(ctx, client.ImageGenerationRequest{
    Model:  client.ModelGLMImage,
    Prompt: "A lighthouse at dusk, oil painting",
})
if err != nil {
    return err
}
result, err := c.WaitForResult(ctx, task.ID, 3*time.Second)
if err != nil {
    return err
}
for _, img := range result.ImageResult {
    fmt.Println(img.URL)
}
```

`VideoGenerationRequest.OffPeak` queues a video task for off-peak processing
at a lower price.

### Multimodal messages (images, video, files)

Set `Message.Images`, `Message.Videos`, or `Message.Files` to attach media.
Each entry is an `https://` URL or a `data:` URI. The client switches the
message's `content` to the content-parts wire shape for you: the text part
first, then one `image_url`, `video_url`, or `file_url` part per entry (the
`Part*` constants name the part types).

```go
resp, err := c.Chat().Create(ctx, client.ChatRequest{
    Model: client.DefaultVisionModel,
    Messages: []client.Message{{
        Role:    "user",
        Content: "What does the chart show, and how does the report explain it?",
        Images:  []string{"https://example.com/chart.png"},
        Files:   []string{"https://example.com/report.pdf"},
        // Videos: []string{"https://example.com/clip.mp4"},
    }},
})
if err != nil {
    return err
}
fmt.Println(resp.Choices[0].Message.Content)
```

Pick a model whose catalog capabilities cover the media you send
(`CapVision`, `CapVideo`, `CapFile`):

```go
entry, ok := client.CatalogEntry(client.DefaultVisionModel)
if ok && slices.Contains(entry.Capabilities, client.CapVideo) {
    // video input is supported
}
```

### Structured output

Z.AI has no `json_schema` response format. Request JSON-object output and put
the schema in the system prompt; `JSONSchemaPrompt` builds that instruction:

```go
schema := json.RawMessage(`{
  "type": "object",
  "properties": {"city": {"type": "string"}, "temp_c": {"type": "number"}},
  "required": ["city", "temp_c"]
}`)
instruction, err := client.JSONSchemaPrompt(schema)
if err != nil {
    return err // the schema is not valid JSON
}
resp, err := c.Chat().Create(ctx, client.ChatRequest{
    Model: client.DefaultModel,
    Messages: []client.Message{
        {Role: "system", Content: instruction},
        {Role: "user", Content: "Current weather in Lisbon?"},
    },
    ResponseFormat: client.JSONObjectFormat(),
})
if err != nil {
    return err
}
var weather struct {
    City  string  `json:"city"`
    TempC float64 `json:"temp_c"`
}
if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &weather); err != nil {
    return err // the model did not return the JSON you asked for
}
```

`JSONObjectFormat()` asks for a JSON object; conformance to the schema comes
from the prompt, so validate the decoded result yourself.

### Function calling

For manual control, inspect `resp.Choices[0].Message.ToolCalls` yourself and
append `role: "tool"` messages before calling `Create` again. For the common
case, `RunWithTools` drives that loop for you:

```go
resp, err := c.Chat().RunWithTools(ctx, req, func(name, arguments string) (string, error) {
    switch name {
    case "get_weather":
        return `{"temp_c": 18}`, nil
    default:
        return "", fmt.Errorf("unknown tool %q", name)
    }
})
if err != nil {
    return err
}
fmt.Println(resp.Choices[0].Message.Content)
```

It executes each tool call, appends the assistant + tool messages (including
the assistant turn's `ReasoningContent`), and repeats until the model returns
a non-tool finish reason or `ToolMaxRounds` (8) is exceeded — use
`RunWithToolsLimit` to set a different cap. A tool executor error is reported
back to the model as the tool's result (`"error: ..."`), not returned to your
caller, so the model can recover instead of the whole exchange failing.

#### Tool types

A `Tool` carries one of four payloads, selected by its `Type`:

| Constructor | `Type` | Payload |
|---|---|---|
| `NewFunctionTool(name, desc, params)` | `ToolTypeFunction` (`"function"`) | `FunctionDef` — a callable the model invokes by name |
| `NewWebSearchTool(engine)` | `ToolTypeWebSearch` (`"web_search"`) | `WebSearchDef` — built-in web search, run server-side |
| `NewRetrievalTool(knowledgeID, promptTemplate)` | `ToolTypeRetrieval` (`"retrieval"`) | `Retrieval` — a knowledge base to ground the answer |
| `NewMCPTool(label, url, allowedTools...)` | `ToolTypeMCP` (`"mcp"`) | `MCPServer` — a remote or Z.AI-hosted MCP server, called server-side |

Only `function` is confirmed against the live API. `web_search`, `retrieval`,
and `mcp` follow docs.z.ai and the official SDKs and are **NOT VERIFIED
LIVE** here; see [Roadmap](roadmap.md).

**web_search.** `NewWebSearchTool` builds an enabled tool that returns its
sources. Tune it through the `WebSearch` payload (`Count` 1–50,
`SearchDomainFilter`, `SearchRecencyFilter`, `ContentSize`, `SearchPrompt`,
`SearchQuery` to force a query). `client.SearchEnginePrime` is the engine for
the global platform; `client.SearchEngines` lists the others, which belong to
the China platform. This is distinct from the standalone
`c.Tools().WebSearch` endpoint.

```go
search := client.NewWebSearchTool(client.SearchEnginePrime)
search.WebSearch.Count = 5
search.WebSearch.SearchRecencyFilter = client.SearchRecencyOneWeek

resp, err := c.Chat().Create(ctx, client.ChatRequest{
    Model:    client.DefaultModel,
    Messages: []client.Message{{Role: "user", Content: "What changed in Go 1.26?"}},
    Tools:    []client.Tool{search},
})
if err != nil {
    return err
}
for _, src := range resp.WebSearch { // the sources the answer grounded in
    fmt.Println(src.Title, src.Link)
}
```

**retrieval.** The knowledge-base product is served by the China platform.
`promptTemplate` may use the `{{knowledge}}` and `{{question}}` placeholders.

```go
kb := client.NewRetrievalTool("your-knowledge-id", "Use {{knowledge}} to answer: {{question}}")
req.Tools = []client.Tool{kb}
```

**mcp.** `ServerLabel` names the server; leave the URL empty to use one of
Z.AI's hosted MCP servers by label, or give a URL for your own. The optional
`allowedTools` restricts which of its tools the model may call. The transport
defaults to streamable HTTP (`MCPTransportStreamableHTTP`);
`MCPTransportSSE` is the alternative. The model's MCP activity comes back in
`ToolCall.MCP` (`MCPCall`: a tool listing or a call with its output or error).

```go
remote := client.NewMCPTool("docs", "https://mcp.example.com/mcp", "search", "fetch")
remote.MCP.TransportType = client.MCPTransportSSE
remote.MCP.Headers = map[string]string{"Authorization": "Bearer " + os.Getenv("DOCS_MCP_TOKEN")}

hosted := client.NewMCPTool("zread", "") // Z.AI-hosted server, chosen by label
req.Tools = []client.Tool{remote, hosted}
```

The client validates these rules before sending, so you get a clear local
error instead of the server's opaque one:

- **Tool-name pattern** — `tools[].function.name` must match
  `^[A-Za-z0-9_-]{1,64}$`.
- **Function cap** — at most `ToolMaxFunctions` (128) function tools per
  request.
- **Per-type payload** — a tool must carry the payload for its type: a
  `function` payload, a `web_search` payload, a `retrieval` payload with a
  `knowledge_id`, or an `mcp` payload with a `server_label`. Unknown types are
  rejected.

#### Response fields worth checking

Beyond `resp.Choices[0].Message.Content`:

- `resp.Choices[0].FinishReason` — compare against the `FinishReason*`
  constants (`FinishReasonStop`, `FinishReasonToolCalls`, `FinishReasonLength`,
  `FinishReasonSensitive`, `FinishReasonModelContextWindowExceeded`,
  `FinishReasonNetworkError`). The last three are documented values that
  signal non-content terminations.
- `resp.Choices[0].Message.ReasoningContent` — the model's reasoning, for
  thinking models.
- `resp.WebSearch` — the top-level `web_search` array the response carries
  when a `web_search` tool fired (each entry's `Link`/`Title`/`Content` are
  the sources the answer grounded in). NOT VERIFIED LIVE.
- `resp.Usage` — token counts, including `CompletionTokensDetails.ReasoningTokens`
  and `PromptTokensDetails.CachedTokens`. `resp.RequestID` carries the
  platform's request ID.

`Pricing.Cost` turns a `Usage` into a dollar figure at a model's catalog
rates (USD per 1M tokens), billing cached prompt tokens at the cached rate.
It is an estimate from the catalog snapshot, not an invoice:

```go
if entry, ok := client.CatalogEntry(req.Model); ok && entry.Pricing != nil {
    fmt.Printf("about $%.6f\n", entry.Pricing.Cost(resp.Usage))
}
```

#### Tool-schema compatibility

GLM's chat endpoint uses a strict JSON-Schema parser for tool `parameters`:
a schema containing `anyOf`, `oneOf`, `allOf`, or a `$ref`/`$defs` reference
makes it return **HTTP 500** rather than a usable error. Those constructs are
exactly what typed languages emit — a nullable field becomes
`anyOf: [{…}, {"type":"null"}]`, a reused struct becomes a `$ref`.

By default the client rewrites tool schemas into the flat subset GLM accepts
before every chat, Anthropic, and Responses request (nullable unions collapse to the underlying type,
`allOf` merges, `$ref` inlines), keeping as much type/description information
as possible. It's a no-op for schemas already in the supported subset and
never mutates your `req.Tools`.

- To normalize a schema yourself (e.g. you build requests elsewhere):
  `client.SanitizeToolSchemas(tools)`.
- To send schemas through untouched (debugging, or a future endpoint that
  supports the full draft): set `Config.DisableToolSchemaCompat = true`.

## Anthropic-compatible Messages API

Z.AI also exposes an Anthropic-protocol surface at `/api/anthropic` — the same
endpoint the GLM Coding Plan points Claude Code at. `c.Anthropic()` is a typed
client for its `POST /v1/messages`, parallel to `c.Chat()` for the OpenAI-style
surface. It calls the region's `Region.AnthropicBaseURL()` (independent of
`Config.BaseURL`), authenticates with your z.ai key as a Bearer token (not
Anthropic's `x-api-key`), and sends an `anthropic-version` header
automatically.

```go
req := client.AnthropicMessageRequest{
    Model:     client.DefaultModel,
    MaxTokens: 1024, // required by the Messages API
    System:    "You are concise.",
    Messages: []client.AnthropicMessage{
        client.AnthropicTextMessage("user", "Explain goroutines in one line"),
    },
}
resp, err := c.Anthropic().Create(ctx, req)
if err != nil {
    return err
}
fmt.Println(resp.Text()) // concatenated text blocks
```

`Stream` returns an iterator over Anthropic's raw SSE events
(`message_start`, `content_block_delta`, …), with the same semantics as
`Chat().Stream`. Each `AnthropicStreamEvent` carries the event name in `Type`
and the JSON payload in `Data`. `Delta()` decodes a `content_block_delta`
event into an `AnthropicDelta` (`Type` is `text_delta`, `thinking_delta`,
`input_json_delta`, or `signature_delta`; `Text`, `Thinking`, and
`PartialJSON` carry the fragment) and reports `ok == false` for any other
event, which you unmarshal from `Data` yourself:

```go
for ev, err := range c.Anthropic().Stream(ctx, req) {
    if err != nil {
        return err // an `event: error` arrives here as an *APIError
    }
    if d, ok := ev.Delta(); ok && d.Type == "text_delta" {
        fmt.Print(d.Text)
    }
}
```

Tools declared via `AnthropicTool.InputSchema` get the same GLM schema
normalization as chat tools (see above). `Config.DisableToolSchemaCompat`
disables it.

Extended thinking (GLM models are reasoning models) is enabled per request and
read back with `resp.Thinking()`:

```go
req.Thinking = &client.AnthropicThinking{Type: "enabled", BudgetTokens: 2048}
resp, err := c.Anthropic().Create(ctx, req)
if err != nil {
    return err
}
fmt.Println(resp.Thinking()) // thinking blocks, or reasoning_content if the
                             // endpoint surfaces reasoning that way instead
fmt.Println(resp.Text())     // the answer, without the reasoning mixed in
```

The success-path response shape is modeled from Anthropic's documented Messages
API and is not yet live-verified here — see [Roadmap](roadmap.md).

## Responses API (Codex protocol)

Z.AI serves the OpenAI Responses protocol at `/api/v1` — the endpoint Codex
is configured against, documented for the GLM Coding Plan
([docs](https://docs.z.ai/devpack/tool/codex)). `c.Responses()` calls
`POST /responses` on `Region.ResponsesBaseURL()` (`https://api.z.ai/api/v1`,
or `https://open.bigmodel.cn/api/v1` for `RegionChina`), independent of
`Config.BaseURL`. The input is a list of items rather than messages, and
`Instructions` plays the role of a system message:

```go
resp, err := c.Responses().Create(ctx, client.ResponsesRequest{
    Model:        client.DefaultModel,
    Instructions: "You are concise.",
    Input:        []client.ResponsesItem{client.ResponsesMessage("user", "Explain goroutines in one line")},
    Reasoning:    &client.ResponsesReasoning{Effort: client.EffortHigh},
})
if err != nil {
    return err
}
fmt.Println(resp.OutputText())    // the message text
fmt.Println(resp.ReasoningText()) // reasoning summaries, or the full reasoning text
```

- `Reasoning.Effort` is validated against the catalog, like
  `ChatRequest.ReasoningEffort`.
- Function tools use the Responses API's flat shape
  (`ResponsesTool{Name, Description, Parameters}`; `Type` defaults to
  `function`) and get the same schema rewrite as chat tools.
  `resp.FunctionCalls()` returns the calls; answer each with
  `client.ResponsesFunctionOutput(call.CallID, output)` appended to `Input`
  in the next request.
- A response whose `Status` is `failed` comes back as an `*APIError`.
- `resp.GetUsage()` maps the token counts onto `client.Usage`, so hooks see
  them like chat usage.

`Stream` yields `ResponsesStreamEvent`s. Text arrives in `Delta` on
`ResponsesEventOutputTextDelta` events, reasoning on
`ResponsesEventReasoningTextDelta` or `ResponsesEventReasoningSummaryDelta`,
and the final response on `ResponsesEventCompleted`; `error` and
`response.failed` events end the stream with an `*APIError`:

```go
for ev, err := range c.Responses().Stream(ctx, req) {
    if err != nil {
        return err
    }
    switch ev.Type {
    case client.ResponsesEventOutputTextDelta:
        fmt.Print(ev.Delta)
    case client.ResponsesEventCompleted:
        fmt.Println("\ntokens:", ev.Response.Usage.TotalTokens)
    }
}
```

NOT VERIFIED LIVE: Z.AI documents the endpoint but not its schema, so the
types follow OpenAI's Responses API; see [Roadmap](roadmap.md).

## Account detection and status

A Z.AI key is either pay-as-you-go or a GLM Coding Plan subscription, and the
two use different chat roots. `Detection().DetectAccountType` classifies the
key without spending anything: it calls the coding-plan-only quota endpoint on
the client's region, then on the other region (a China-issued coding-plan key
answers only on open.bigmodel.cn). A well-formed quota response confirms a
coding plan; if neither gateway identifies a subscription but at least one
answered, the key is treated as pay-as-you-go. If neither gateway can be
reached, the transport error is returned instead of a guess. The result is
cached per client.

```go
acct, err := c.Detection().DetectAccountType(ctx)
if err != nil {
    return err
}
fmt.Println(acct.Type)      // client.AccountTypeCodingPlan or client.AccountTypePayAsYouGo
fmt.Println(acct.Region)    // the gateway the key belongs to
fmt.Println(acct.BaseURL)   // the chat root to use (Region.BaseURLFor)
fmt.Println(acct.Confirmed) // see below
fmt.Println(acct.Level)     // coding-plan tier (lite, pro, max); empty for pay-as-you-go
```

`Confirmed` is true only for a coding plan, which the quota endpoint
positively identifies. No endpoint identifies a pay-as-you-go key, so that
result is an inference by elimination and is never confirmed.

`CheckAccountStatus` answers "does this key work and can it spend right now?":

```go
st, err := c.Detection().CheckAccountStatus(ctx)
if err != nil {
    return err
}
fmt.Println(st.APIAccessible, st.HasBalance, st.Message)
```

For a coding-plan key it is a free quota check (`HasBalance` is false when a
model-usage window is exhausted). Z.AI has no documented balance call for
pay-as-you-go keys, so for those it sends one billed one-token completion on
`client.BalanceProbeModel` and classifies the outcome (works, no balance, rate
limited, bad key). Call it on demand, never on a timer.

Quota windows come from `c.Quota()`; `QuotaData.Exhausted`,
`QuotaLimit.IsModelLimit`, and `QuotaLimit.UsedFraction` interpret both the
credit-based and the legacy token windows (see [Accounts &
Quota](accounts-and-quota.md)):

```go
q, err := c.Quota().GetQuotaLimit(ctx)
if err != nil {
    return err
}
for _, l := range q.Data.Limits {
    fmt.Printf("%s: %.0f%% used\n", l.WindowDescription(), l.UsedFraction()*100)
}
```

### Balance and subscriptions

NOT VERIFIED LIVE. `Account().Balance` and `Account().Subscriptions` call the
biz API (`Region.BizBaseURL()`), which Z.AI does not document; the routes and
field names come from Z.AI's own ZCode client and community usage tools (see
[Roadmap](roadmap.md)). A business failure reported inside an HTTP 200 comes
back as an `*APIError`.
Balance amounts are in the region's billing currency (USD on api.z.ai, CNY on
open.bigmodel.cn); the API sends no currency field.

```go
bal, err := c.Account().Balance(ctx)
if err != nil {
    return err
}
fmt.Printf("available %.2f of %.2f\n", bal.AvailableBalance, bal.Balance)

subs, err := c.Account().Subscriptions(ctx) // empty when the account has none
if err != nil {
    return err
}
for _, s := range subs {
    fmt.Println(s.ProductName, s.Status, s.NextRenewTime)
}
```

## Error handling

See [Error Handling](error-handling.md) for the full `APIError` reference,
error codes, in-band stream errors, and the retry behavior you get by default.

## Observability hooks

`Config.Hooks` attaches observability hooks (tracing, metrics, logging) that
fire on every request, response, error, and stream chunk — without you
wrapping the `http.RoundTripper`. The interface is stdlib-only so `pkg/client`
stays dependency-free; concrete implementations live in `pkg/observe`
(OpenTelemetry) or you can write your own.

```go
import (
    "github.com/SamyRai/go-z-ai/pkg/client"
    "github.com/SamyRai/go-z-ai/pkg/observe"
)

c, err := client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
    Hooks:  []client.Hook{observe.NewOTelHook("my-service")},
})
```

A `Hook` fires:

- `OnRequest(ctx, meta) context.Context` — per attempt, before the HTTP send;
  the returned context replaces the input (attach tracing spans here).
- `OnResponse(ctx, meta)` — when an attempt succeeds (a 2xx response was
  parsed, or a stream ended cleanly), with status code, duration, and token
  usage (when the response carries one).
- `OnError(ctx, meta, err)` — when an attempt fails: a transport error, a
  non-2xx response (including one that is about to be retried), an
  undecodable body, a mid-stream failure, or a stream you stopped ranging over
  early (`context.Canceled`). On the last attempt `err` is the error the caller
  sees.
- `OnStreamChunk(ctx, meta, chunk)` — for each chunk from `Chat().Stream`,
  `Anthropic().Stream`, or `Responses().Stream`; `chunk` is a
  `client.StreamChunk`, a `client.AnthropicStreamEvent`, or a
  `client.ResponsesStreamEvent`.

Every `OnRequest` is followed by exactly one `OnResponse` or `OnError` for the
same attempt, so a span started in `OnRequest` can always be ended there.

`RequestMeta` carries `Service`, `Method`, `Endpoint`, `Model`, and `Attempt`
fields. Most services stamp `Service` (and `Model`, where the request carries
one) into the request automatically — chat, anthropic, embeddings, audio,
files, quota, and others; the rest leave them empty unless you stamp them
yourself via `client.WithService(ctx, "...")` / `client.WithModel(ctx, "...")`
before the call. The context values also override what a service sets.

A nil/empty `Hooks` slice skips all invocation — the no-hook path is
zero-allocation and has no measurable overhead.

## Multi-account credential management

The multi-account credential store and the GLM Coding Plan credential/config
writers live in `internal/accounts` and `internal/coding`. They are internal
to this module — not part of the importable public API — so they can evolve
without semver constraints. The public packages are `pkg/client` and the
optional `pkg/observe`; the `accounts` and `coding` CLI commands are the
stable way to drive that functionality. (These packages lived under `pkg/` before and were importable;
see the [CHANGELOG](../../CHANGELOG.md) for the move.)

## Testing your own code against this client

Every service method is a plain function on an interface-free concrete type,
so the usual Go approach is to point `Config.BaseURL` at an `httptest.Server`
you control. If you want to replay *real* recorded Z.AI traffic instead of a
hand-written stub, see how this repo's own tests do it with
[go-vcr](https://github.com/dnaeon/go-vcr) — `pkg/client/*_test.go` and
`pkg/client/testdata/cassettes/` — and read
[Contributing § the live-verification convention](../../CONTRIBUTING.md) for why.

## Architecture notes

For how the services are structured internally (the request facade,
retry and timeout design, why some services hit a different host) see
[Architecture](architecture.md).
