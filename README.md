# Z.AI API Client

**English** | [简体中文](README.zh.md) | [Русский](README.ru.md) | [Deutsch](README.de.md) | [Татарча](README.tt.md) | [Türkçe](README.tr.md)

[![CI](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SamyRai/go-z-ai.svg)](https://pkg.go.dev/github.com/SamyRai/go-z-ai)
[![OpenSSF Scorecard](https://img.shields.io/ossf-scorecard/github.com/SamyRai/go-z-ai?label=openssf%20scorecard)](https://securityscorecards.dev/view.html?uri=github.com/SamyRai/go-z-ai)
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
**[Getting Started →](docs/en/getting-started.md)**

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

More runnable programs — streaming, async image polling, the Anthropic
`/v1/messages` endpoint — live under [`examples/`](examples/).

## Documentation

**[Full documentation index →](docs/en/README.md)**

| | |
|---|---|
| [Getting Started](docs/en/getting-started.md) | [CLI Reference](docs/en/cli-reference.md) |
| [Accounts & Quota](docs/en/accounts-and-quota.md) | [Coding Tools](docs/en/coding-tools.md) |
| [Library Guide](docs/en/library-guide.md) | [Error Handling](docs/en/error-handling.md) |
| [Architecture](docs/en/architecture.md) | [Roadmap & Known Limitations](docs/en/roadmap.md) |
| [Contributing](CONTRIBUTING.md) | [Security Policy](SECURITY.md) |
| [Code of Conduct](CODE_OF_CONDUCT.md) | [Changelog](CHANGELOG.md) |

## What's covered

Chat (streaming, structured output, deep thinking, function calling, vision),
the Anthropic-compatible `/v1/messages` endpoint, Models, Images, Video, Audio
(transcription + TTS + voice cloning), OCR & document parsing, Embeddings,
Moderations, Rerank, Agents, Files, Batch jobs, GLM Coding Plan
usage/quota/multi-account management, and a full-screen terminal UI
(`zai-client tui`). See [CLI Reference](docs/en/cli-reference.md) for the complete
command list or [Library Guide](docs/en/library-guide.md) for the Go API.

## How it relates to the official SDKs

Z.AI / Zhipu publish official SDKs for **Python**
([zai-org/z-ai-sdk-python](https://github.com/zai-org/z-ai-sdk-python), PyPI
`zai-sdk`), **Node** ([MetaGLM/zhipuai-sdk-nodejs-v4](https://github.com/MetaGLM/zhipuai-sdk-nodejs-v4)),
and **Java** ([MetaGLM/zhipuai-sdk-java-v4](https://github.com/MetaGLM/zhipuai-sdk-java-v4)).
There is **no official Go SDK** — `go-z-ai` fills that gap, and layers a CLI,
a TUI, regional gateway switching (`api.z.ai` ↔ `open.bigmodel.cn`), and GLM
Coding Plan multi-account management on top of the same API surface.

> ℹ️ `zai-claude-config.json` at the repo root is a **template** with
> placeholder values (`"your-zai-api-key-here"`) used by
> `zai-client coding load claude-code`. It is not a real config and ships no
> credentials.

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
