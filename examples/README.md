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
| [`chat-streaming`](chat-streaming) | Token-by-token SSE streaming via `Chat().CreateStream` (callback-based). |
| [`async-poll`](async-poll) | Async image generation: `Images().GenerateAsync` then `WaitForResult`. |
| [`anthropic-messages`](anthropic-messages) | The Anthropic-compatible `/v1/messages` endpoint via `Anthropic().Create`. |

These are deliberately small — for the full surface (tools, vision, structured
output, batch, voice, the CLI, the TUI) see the [Library Guide](../docs/library-guide.md)
and [CLI Reference](../docs/cli-reference.md).

## Notes

- Streaming uses a callback (`func(StreamChunk) error`), not a channel or
  iterator. Return a non-nil error from the callback to abort the stream.
- Async tasks start in `PROCESSING` and end in `SUCCESS` or `FAIL`. URL outputs
  expire after ~30 days.
- The Anthropic endpoint authenticates with a Bearer token (the same
  `ZAI_API_KEY`), not the Anthropic `x-api-key` header. The
  `anthropic-version: 2023-06-01` header is added automatically.
