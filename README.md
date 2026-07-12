# Z.AI API Client

[![CI](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SamyRai/go-z-ai.svg)](https://pkg.go.dev/github.com/SamyRai/go-z-ai)
[![Go Report Card](https://goreportcard.com/badge/github.com/SamyRai/go-z-ai)](https://goreportcard.com/report/github.com/SamyRai/go-z-ai)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A Go CLI and client library for the Z.AI (Zhipu AI) API platform: chat
completions, models, images, video, audio, embeddings, moderation, rerank,
agents, batch jobs, file parsing, GLM Coding Plan account/quota management,
and a Go port of `@z_ai/coding-helper` for wiring Claude Code, OpenCode,
Crush, Factory Droid, and Cursor to your GLM Coding Plan.

## Install

```bash
go install github.com/SamyRai/go-z-ai@latest
```

Requires Go 1.26.4+ and a [Z.AI API key](https://z.ai/manage-apikey).
Building from source, first-run auth, and troubleshooting:
**[Getting Started →](docs/getting-started.md)**

## Quick example

```bash
export ZAI_API_KEY=your_api_key_here
zai-client chat create "Explain goroutines in one paragraph" --stream
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

c, _ := client.NewClientFromEnv()
resp, _ := c.Chat().Create(ctx, client.ChatRequest{
    Model:    "glm-5.2",
    Messages: []client.Message{{Role: "user", Content: "Explain goroutines in one paragraph"}},
})
fmt.Println(resp.Choices[0].Message.Content)
```

## Documentation

**[Full documentation index →](docs/README.md)**

| | |
|---|---|
| [Getting Started](docs/getting-started.md) | [CLI Reference](docs/cli-reference.md) |
| [Accounts & Quota](docs/accounts-and-quota.md) | [Coding Tools](docs/coding-tools.md) |
| [Library Guide](docs/library-guide.md) | [Error Handling](docs/error-handling.md) |
| [Architecture](docs/architecture.md) | [Contributing](CONTRIBUTING.md) |

## What's covered

Chat (streaming, structured output, deep thinking, function calling, vision),
Models, Images, Video, Audio (transcription + TTS + voice cloning), OCR &
document parsing, Embeddings, Moderations, Rerank, Agents, Files, Batch jobs,
GLM Coding Plan usage/quota/multi-account management, and a full-screen
terminal UI (`zai-client tui`). See [CLI Reference](docs/cli-reference.md) for
the complete command list or [Library Guide](docs/library-guide.md) for the
Go API.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) — in particular, this project's
live-verification convention (recorded API cassettes instead of hand-wished
fixtures) if you're adding or changing a service.

## License

Apache License 2.0 — see [LICENSE](LICENSE).

## Support

- **Z.AI API docs**: [https://docs.z.ai](https://docs.z.ai)
- **Issues**: [GitHub Issues](https://github.com/SamyRai/go-z-ai/issues)
- **Security**: see [SECURITY.md](SECURITY.md) — please don't file vulnerabilities as public issues
