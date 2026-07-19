# go-z-ai

A Go **CLI**, **library**, and **TUI** for the Z.AI (Zhipu AI / BigModel)
platform — every GLM model surface in one tool, plus a Go port of
`@z_ai/coding-helper` that wires Claude Code, OpenCode, Crush, Factory Droid,
and Cursor to your GLM Coding Plan.

**English** | [简体中文](README.zh.md) | [Русский](README.ru.md) | [Deutsch](README.de.md) | [Татарча](README.tt.md) | [Türkçe](README.tr.md)

[![CI](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SamyRai/go-z-ai.svg)](https://pkg.go.dev/github.com/SamyRai/go-z-ai)
[![OpenSSF Scorecard](https://img.shields.io/ossf-scorecard/github.com/SamyRai/go-z-ai?label=openssf%20scorecard)](https://securityscorecards.dev/viewer/?uri=github.com/SamyRai/go-z-ai)
[![Latest release](https://img.shields.io/github/v/release/SamyRai/go-z-ai)](https://github.com/SamyRai/go-z-ai/releases)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

## Quick example

```bash
# 1. Configure (any of these works — env var, .env file, or --config <file>)
export ZAI_API_KEY=your_api_key_here
# or: cp .env.example .env  &&  edit .env

# 2. Use the CLI
go-z-ai chat create "Explain goroutines in one paragraph" --stream
```

```go
// …or import the library — no CLI required.
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

## Features

- **Chat** — streaming, structured output (JSON Schema), deep thinking,
  function/tool calling, vision (`glm-4.6v`/`glm-4.5v`), and an
  **Anthropic-compatible `/v1/messages`** endpoint (the same one Claude Code
  and Cursor hit when wired to a GLM Coding Plan).
- **Media** — image generation, video generation (always async), audio
  transcription, TTS, and GLM-TTS voice cloning.
- **Document understanding** — layout OCR, handwriting OCR, and a document
  parser for RAG preprocessing.
- **Retrieval** — embeddings, rerank, built-in web search / web reader /
  tokenizer tools.
- **Moderations** — content moderation via the China-platform endpoint.
- **Agents** — Z.AI's specialized agents (translation, slide/poster
  generation, video effects).
- **Batch & files** — JSONL batch jobs for chat completions, file
  upload/list/download.
- **GLM Coding Plan** — quota/usage monitoring, multi-account management,
  and `go-z-ai coding` to wire Claude Code, OpenCode, Crush, Factory
  Droid, and Cursor to your subscription.
- **DX** — full-screen terminal UI (`go-z-ai tui`), regional gateway
  switching (`api.z.ai` ↔ `open.bigmodel.cn`), automatic retry with
  backoff + jitter, and a typed `APIError` with every Z.AI error code mapped.

## Install

```bash
go install github.com/SamyRai/go-z-ai@latest
```

This produces a binary named `go-z-ai` on your `$GOPATH/bin`.

```bash
# Optional short alias: ln -s "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai"
```

Requires Go 1.26.4+ and a [Z.AI API key](https://z.ai/manage-apikey/apikey-list).
Building from source, first-run auth, and troubleshooting:
**[Getting Started →](docs/en/getting-started.md)**

## As a CLI

A single `go-z-ai` binary covering the full surface. Every command
supports `--help`; the quick tour:

```bash
go-z-ai chat create "..." --stream          # chat (streaming, tools, vision, structured output)
go-z-ai anthropic messages "..." --stream   # Anthropic-compatible /v1/messages
go-z-ai image|video|audio|voice ...         # media generation, transcription, TTS, cloning
go-z-ai ocr|parser ...                      # OCR + document parsing
go-z-ai embeddings|rerank|moderations ...   # retrieval + content moderation
go-z-ai models list                         # model catalog + pricing
go-z-ai accounts add|use|quota|usage ...    # multi-account + GLM Coding Plan monitoring
go-z-ai coding auth|load|doctor|mcp ...     # wire Claude Code / Cursor / etc. to GLM Coding Plan
go-z-ai tui                                 # full-screen terminal UI (all of the above)
go-z-ai validate                            # confirm your key works with one real call
```

Every result-producing command takes `--format text|json` (JSON goes to
stdout, progress chatter to stderr, so you can pipe into `jq`).

→ Full command list: **[CLI Reference](docs/en/cli-reference.md)**

## As a Go library

`pkg/client` is the only public importable package; everything under
`internal/` is implementation detail. Retry, timeout, regional gateway
selection, and error mapping are centralized — services never build their own
`http.Client` or issue raw requests.

```bash
go get github.com/SamyRai/go-z-ai
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

c, err := client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
    // Optional: BaseURL, Timeout, MaxRetries, RetryDelay, ChinaAPIKey, Region
})
```

Services, all following `c.<Service>().<Method>(ctx, …)`:

| Accessor | Covers |
|---|---|
| `c.Chat()` | Completions, streaming, async, `RunWithTools` |
| `c.Anthropic()` | Anthropic-protocol `/v1/messages` (Create, CreateStream) |
| `c.Models()` | List, Get, text/vision/free filters |
| `c.Images()` / `c.Videos()` | Image (sync/async), video (always async) |
| `c.Audio()` / `c.Voice()` | Transcription, TTS, voice cloning |
| `c.Layout()` / `c.FileParser()` | OCR + document-to-text for RAG |
| `c.Files()` / `c.Batch()` | Upload, batch jobs |
| `c.Agents()` | Z.AI specialized agents |
| `c.Embeddings()` / `c.Rerank()` / `c.Moderations()` | Retrieval + moderation |
| `c.Tools()` | WebSearch, WebReader, Tokenize |
| `c.Usage()` / `c.Quota()` / `c.Account()` / `c.Detection()` | GLM Coding Plan monitoring |
| `c.GetAsyncResult()` / `c.WaitForResult()` | Shared polling for async tasks |

→ Full API with examples: **[Library Guide](docs/en/library-guide.md)**
→ Generated reference: [pkg.go.dev](https://pkg.go.dev/github.com/SamyRai/go-z-ai)

## Configuration

Three ways to provide credentials, resolved in this priority order
(highest wins):

| Method | When to use |
|---|---|
| `--api-key <key>` flag | One-off calls, scripts, CI |
| `--account <name>` flag | Switch between [stored accounts](docs/en/accounts-and-quota.md) |
| `ZAI_API_KEY` env var (or `.env` file) | Everyday local shell use |
| Accounts store's active account | After `go-z-ai accounts use <name>` |

The `.env` file is the common case — copy the annotated template and edit it:

```bash
cp .env.example .env
# or point at any file: go-z-ai --config /path/to/config ...
```

```dotenv
ZAI_API_KEY=your_api_key_here
# ZAI_API_BASE_URL=https://api.z.ai/api/paas/v4     # override the chat endpoint
# ZAI_REGION=china                                   # if your key was issued on open.bigmodel.cn
# ZAI_CHINA_API_KEY=...                              # separate bigmodel.cn credential
# ZAI_ENV=production
```

→ Full reference (multi-account, regional gateways, quota windows):
**[Accounts & Quota](docs/en/accounts-and-quota.md)**

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
> `go-z-ai coding load claude-code`. It is not a real config and ships no
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
