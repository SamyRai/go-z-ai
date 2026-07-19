# Examples

Minimal, runnable Go programs that exercise the `pkg/client` public API. All
read the API key from the environment (`ZAI_API_KEY`), so set it first:

```bash
export ZAI_API_KEY=your_api_key_here
```

Each example lives in its own `package main` inside the main module, so you can
run it with `go run ./examples/<name>` and CI's `go build ./...` compiles them
on every push.

| Example | What it shows |
|---|---|
| [`chat-streaming`](chat-streaming) | Token-by-token SSE streaming via `Chat().Stream` (Go 1.23+ iterator). |
| [`async-poll`](async-poll) | Async image generation: `Images().GenerateAsync` then `WaitForResult`. |
| [`anthropic-messages`](anthropic-messages) | The Anthropic-compatible `/v1/messages` endpoint via `Anthropic().Create`. |
| [`observability`](observability) | Wiring an OpenTelemetry hook (`pkg/observe`) so every API call emits a span carrying the GenAI semantic-convention attributes. |

These are deliberately small — for the full surface (tools, vision, structured
output, batch, voice, the CLI, the TUI) see the [Library Guide](../docs/en/library-guide.md)
and [CLI Reference](../docs/en/cli-reference.md).

## Notes

- Streaming uses a Go 1.23+ iterator (`for chunk, err := range c.Chat().Stream(ctx, req)`).
  The older callback-based `CreateStream` is deprecated and delegates to `Stream`.
- Async tasks start in `PROCESSING` and end in `SUCCESS` or `FAIL`. URL outputs
  expire after ~30 days.
- The Anthropic endpoint authenticates with a Bearer token (the same
  `ZAI_API_KEY`), not the Anthropic `x-api-key` header. The
  `anthropic-version: 2023-06-01` header is added automatically.
- The `observability` example pulls OpenTelemetry SDK + a stdout exporter as
  deps; the rest stay stdlib-only. All examples compile under CI's
  `go build ./...`.
