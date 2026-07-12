# Library Guide

`pkg/client` is a standalone Go library — everything the CLI does, it does by
calling this package. You can depend on it directly without the CLI at all.

```bash
go get github.com/SamyRai/go-z-ai
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"
```

## Creating a client

```go
c, err := client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
})
```

Or, if you just want the env-var default with no other config:

```go
c, err := client.NewClientFromEnv() // reads ZAI_API_KEY, ZAI_API_BASE_URL
```

`Config` fields:

| Field | Default | Notes |
|---|---|---|
| `APIKey` | — | Required |
| `BaseURL` | `https://api.z.ai/api/paas/v4` | Override for the coding-plan endpoint, etc. |
| `HTTPClient` | an internally configured `*http.Client` | Bring your own transport if you need custom TLS/proxy behavior |
| `Timeout` | 30s | Bounds dial/TLS/response-header wait — **not** the whole response body read, so it never truncates a live SSE stream |
| `MaxRetries` | 3 | Retries on 429/5xx/network errors. `-1` disables retries entirely |
| `RetryDelay` | 200ms | Base exponential-backoff delay |
| `ChinaAPIKey` | falls back to `APIKey` | Only needed if you hold a separate bigmodel.cn-only credential — see [Accounts & Quota](accounts-and-quota.md#china-platform-key) |

Every service method takes `context.Context` as its first argument and
propagates it all the way to the HTTP call — cancel it to abort a request or
a pending retry backoff.

## Services

`Client` exposes one method per service, all following the same
`c.<Service>().<Method>(ctx, ...)` shape:

| Accessor | Covers |
|---|---|
| `c.Chat()` | Completions — `Create`, `CreateAsync`, `CreateStream`, `CreateSimple`, `RunWithTools` |
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
| `c.Embeddings()` | `Create` (routes to `open.bigmodel.cn`) |
| `c.Moderations()` | `Create` (routes to `open.bigmodel.cn`) |
| `c.Rerank()` | `Create` |
| `c.Tools()` | `WebSearch`, `WebReader`, `Tokenize` |
| `c.Usage()`, `c.Quota()`, `c.Detection()`, `c.Account()` | GLM Coding Plan usage/quota/account monitoring |
| `c.GetAsyncResult(ctx, id)`, `c.WaitForResult(ctx, id, interval)` | Shared polling for async image/video/chat tasks |

Every request-validation check (required fields, etc.) happens client-side
before a request is sent — you get a local `error` immediately rather than a
round trip for something like a missing `model`.

## Chat completions

```go
resp, err := c.Chat().Create(ctx, client.ChatRequest{
    Model: "glm-5.2",
    Messages: []client.Message{
        {Role: "system", Content: "You are a helpful assistant."},
        {Role: "user", Content: "Explain goroutines in one paragraph"},
    },
    Temperature: 0.7,
})
fmt.Println(resp.Choices[0].Message.Content)
```

### Streaming

```go
err := c.Chat().CreateStream(ctx, req, func(chunk client.StreamChunk) error {
    if len(chunk.Choices) > 0 {
        fmt.Print(chunk.Choices[0].Delta.Content)
    }
    return nil // a non-nil return aborts the stream
})
```

### Async

```go
task, _ := c.Chat().CreateAsync(ctx, req)
result, err := c.WaitForResult(ctx, task.ID, 3*time.Second)
```

### Vision (images in a message)

```go
req.Messages[len(req.Messages)-1].Images = []string{
    "https://example.com/photo.jpg", // or a data: URI
}
req.Model = "glm-4.6v"
```

### Structured output

```go
req.ResponseFormat = client.NewJSONSchemaFormat("my_schema", rawJSONSchema, true /* strict */)
```

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
```

It executes each tool call, appends the assistant + tool messages, and
repeats until the model returns a non-tool finish reason or
`ToolMaxRounds` (8) is exceeded — use `RunWithToolsLimit` to set a different
cap. A tool executor error is reported back to the model as the tool's
result (`"error: ..."`), not returned to your caller, so the model can
recover instead of the whole exchange failing.

## Error handling

See [Error Handling](error-handling.md) for the full `APIError` reference,
error codes, and the retry behavior you get by default.

## Multi-account credential management

If you're building something that manages multiple Z.AI accounts (like the
CLI's `accounts` command does), `pkg/accounts` is reusable on its own:

```go
import "github.com/SamyRai/go-z-ai/pkg/accounts"

store, err := accounts.Load()
acct, ok := store.Get("personal")
baseURL, err := acct.ResolvedBaseURL() // derives the right endpoint from acct.Type
```

`pkg/coding` is similarly standalone if you're building tooling around the
GLM Coding Plan credential file (`~/.chelper/config.yaml`) or the supported
coding-tool config formats (Claude Code, OpenCode, Crush, Factory Droid,
Cursor) — see [Coding Tools](coding-tools.md) for what it does from the CLI
side; the package API mirrors those same operations (`coding.Load`,
`coding.Unload`, `coding.Detect`, one function per tool plus dispatch-by-ID
variants).

## Testing your own code against this client

Every service method is a plain function on an interface-free concrete type,
so the usual Go approach is to point `Config.BaseURL` at an `httptest.Server`
you control. If you want to replay *real* recorded Z.AI traffic instead of a
hand-written stub, see how this repo's own tests do it with
[go-vcr](https://github.com/dnaeon/go-vcr) — `pkg/client/*_test.go` and
`pkg/client/testdata/cassettes/` — and read
[Contributing § the live-verification convention](../CONTRIBUTING.md) for why.

## Architecture notes

For how the services are structured internally (the `doRequest` facade,
retry/backoff design, why some services authenticate against a different base
URL) see [Architecture](architecture.md).
