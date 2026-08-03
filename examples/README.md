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
| [`quickstart-chat`](quickstart-chat) | Minimal one-shot chat: one message in, the assistant reply out. |
| [`quickstart-structured`](quickstart-structured) | Asking the model for typed data and parsing the response into a Go struct (JSON Schema response format). |
| [`quickstart-vision`](quickstart-vision) | Sending an image URL to a vision-capable model (`glm-4.6v`). |
| [`chat-streaming`](chat-streaming) | Token-by-token SSE streaming via `Chat().Stream` (Go 1.23+ iterator). |
| [`chat-tools`](chat-tools) | Function/tool calling: define a tool, let the model decide to call it, and return the result. |
| [`embeddings-batch`](embeddings-batch) | Generating embeddings for a batch of texts and computing cosine-similarity against a query. |
| [`rerank-documents`](rerank-documents) | Reranking documents with GLM's rerank API (a RAG second stage). |
| [`audio-tts`](audio-tts) | Text-to-speech via `Audio().Speech`, writing the audio bytes to a file. |
| [`async-poll`](async-poll) | Async image generation: `Images().GenerateAsync` then `WaitForResult`. |
| [`quota-usage`](quota-usage) | Reading quota limits and per-model/tool usage (the endpoints behind the web dashboard). |
| [`anthropic-messages`](anthropic-messages) | The Anthropic-compatible `/v1/messages` endpoint via `Anthropic().Create`. |
| [`observability`](observability) | Wiring an OpenTelemetry hook (`pkg/observe`) so every API call emits a span carrying the GenAI semantic-convention attributes. |

These are deliberately small — for the full surface (the CLI, the TUI, the
remaining endpoints) see the [Library Guide](../docs/en/library-guide.md)
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
