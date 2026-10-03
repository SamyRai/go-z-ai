# go-z-ai

A Go **CLI**, **library**, and **TUI** for the Z.AI (Zhipu AI / BigModel)
platform — every GLM model surface in one tool, plus a Go port of
`@z_ai/coding-helper` that wires Claude Code, Codex, OpenCode, Crush, and Factory
Droid to your GLM Coding Plan.

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
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

func main() {
	c, err := client.NewClientFromEnv() // reads ZAI_API_KEY
	if err != nil {
		log.Fatal(err)
	}
	req := client.ChatRequest{
		Model:    client.DefaultModel,
		Messages: []client.Message{{Role: "user", Content: "Explain goroutines in one paragraph"}},
	}
	for chunk, err := range c.Chat().Stream(context.Background(), req) {
		if err != nil {
			log.Fatal(err)
		}
		if len(chunk.Choices) > 0 {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}
	fmt.Println()
}
```

More runnable programs — tool calling, structured output, vision, async image
polling, the Anthropic `/v1/messages` endpoint — live under
[`examples/`](examples/).

## Features

- **Chat** — streaming (a Go iterator in the library), reasoning control
  (`reasoning_effort`, thinking on/off), function/tool calling (plus the
  built-in `web_search`, `retrieval`, and `mcp` tool types), JSON-object output
  with the schema in the prompt, multimodal input (image, video, file), and an
  **Anthropic-compatible `/v1/messages`** endpoint (the same one Claude Code
  hits when wired to a GLM Coding Plan).
- **Models** — a curated catalog (context window, max output, prices,
  capabilities, accepted reasoning efforts) behind `go-z-ai models` and
  `client.CatalogEntry`; defaults are listed [below](#default-models).
- **Media** — image generation, video generation (always async, with optional
  off-peak queueing), audio transcription, TTS, and GLM-TTS voice cloning.
- **Document understanding** — layout OCR, handwriting OCR, and a document
  parser for RAG preprocessing.
- **Retrieval** — embeddings, rerank, built-in web search / web reader /
  tokenizer tools.
- **Moderations** — content moderation via the China-platform endpoint.
  Embeddings and moderations are served by `open.bigmodel.cn` and can be gated
  by account entitlement; see
  [Roadmap & Known Limitations](docs/en/roadmap.md).
- **Agents** — Z.AI's specialized agents (translation, slide/poster
  generation, video effects).
- **Batch & files** — JSONL batch jobs for chat completions, file
  upload/list/download/delete.
- **GLM Coding Plan** — account-type detection, credit-based quota/usage
  monitoring, multi-account management, and `go-z-ai coding` to wire Claude
  Code, Codex, OpenCode, Crush, and Factory Droid to your subscription and register
  Z.AI's four official MCP servers (Vision, web search, web reader, Zread).
- **DX** — full-screen terminal UI (`go-z-ai tui`: chat, models, usage,
  accounts, coding, media, and tools tabs), one `--region` switch
  (`api.z.ai` ↔ `open.bigmodel.cn`) that selects every endpoint, automatic
  retry with backoff + jitter (honoring `Retry-After`), and a typed `APIError`
  with Z.AI error codes mapped (unknown codes fall back to the HTTP status).

### Default models

Callers pick defaults from `pkg/client` constants rather than hard-coding IDs:

| Constant | Model | Used for |
|---|---|---|
| `client.DefaultModel` | `glm-5.3` | Chat, Anthropic endpoint, tokenizer. 1M context, reasoning effort `low`/`high`/`max` |
| `client.DefaultFastModel`, `client.DefaultVisionModel` | `glm-5.3-flash` | Fast, low-cost tier; natively multimodal (image, video, file input) |
| `client.DefaultOCRModel` | `glm-ocr` | `ocr`, `Layout().Parse` |
| `client.DefaultASRModel` | `glm-asr-2512` | `audio transcribe` (clips up to 30 s) |
| `client.DefaultTTSModel` | `glm-tts` | `audio speech` |
| `client.ModelGLMImage`, `client.ModelCogView4` | `glm-image` (default), `cogview-4-250304` | `image generate` |
| `client.VideoModels` | `cogvideox-3` (default), `viduq1-*`, `vidu2-*` | `video generate` |

The catalog also covers `glm-5.3-flashx`, `glm-5.2`, the GLM-4.x family, and the
free tiers (`go-z-ai models free`). It is a curated snapshot (last refreshed
2026-10-02) — `go-z-ai models list` shows context, max output, prices,
capabilities, and effort levels, and live `/models` values win over catalog
values.

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
go-z-ai chat create "..." --stream          # chat (streaming, reasoning effort, tools, image/video/file input, JSON output)
go-z-ai anthropic messages "..." --stream   # Anthropic-compatible /v1/messages
go-z-ai responses create "..." --stream     # OpenAI Responses protocol (/api/v1, what Codex uses)
go-z-ai image|video|audio|voice ...         # media generation, transcription, TTS, cloning
go-z-ai ocr|parser ...                      # OCR + document parsing
go-z-ai embeddings|rerank|moderations ...   # retrieval + content moderation
go-z-ai models list                         # model catalog: context, prices, capabilities, efforts
go-z-ai account detect|status               # key type (coding plan / pay-as-you-go), region, health
go-z-ai accounts add|use|quota|usage ...    # multi-account + GLM Coding Plan monitoring
go-z-ai usage quota                         # GLM Coding Plan quota windows for the current key
go-z-ai coding auth|load|doctor|mcp ...     # wire Claude Code / Codex / OpenCode / Crush / Factory Droid to GLM Coding Plan
go-z-ai tui                                 # full-screen terminal UI (all of the above)
go-z-ai validate                            # confirm your key works (a free request)
```

Most commands that print results take `--format text|json` (JSON goes to
stdout, progress chatter to stderr, so you can pipe into `jq`). The root
`--region global|china` flag (env `ZAI_REGION`) selects every endpoint;
`--base-url` overrides only the chat/PaaS root.

### Coding tools

`go-z-ai coding` is a Go port of Z.AI's `@z_ai/coding-helper`. Supported tools:
`claude-code`, `codex`, `opencode`, `crush`, and `factory-droid`.

```bash
go-z-ai coding auth glm_coding_plan_global <key>   # validate + store the plan key (or glm_coding_plan_china)
go-z-ai coding load claude-code                    # write the tool's config (also: codex, opencode, crush, factory-droid)
go-z-ai coding mcp add claude-code                 # register the official MCP servers (--server to pick)
go-z-ai coding doctor                              # health check; exits non-zero on problems
```

- For Claude Code, the haiku tier maps to `client.DefaultFastModel` and the
  sonnet/opus tiers to `client.DefaultModel`, with Claude Code's `[1m]`
  1M-context suffix. `--haiku`, `--sonnet`, `--opus`, `--no-model-mapping`,
  `--auto-compact-window`, `--max-thinking-tokens`, and `--max-output-tokens`
  tune this on `coding load` and `coding auth`.
- `coding mcp add|remove <tool>` registers Z.AI's four official MCP servers:
  `zai-mcp-server` (Vision; runs locally via `npx`, needs Node.js),
  `web-search-prime`, `web-reader`, and `zread` (hosted, authenticated with the
  plan key). The default is every server the tool supports — Codex takes only
  Vision (openai/codex#14793); `coding mcp status` shows what each tool has.
  MCP calls draw plan quota.
- For Codex, `coding load codex` writes a ZAI provider speaking the Responses
  protocol (`/api/v1`) to `~/.codex/config.toml`, and the model's metadata to
  `~/.codex/models.json`.

→ Full command list: **[CLI Reference](docs/en/cli-reference.md)**

## As a Go library

`pkg/client` is the library (standard library only); `pkg/observe` is an
optional OpenTelemetry hook adapter. Everything under `internal/` is
implementation detail. Retry, timeout, regional gateway selection, and error
mapping are centralized — services never build their own `http.Client` or issue
raw requests.

```bash
go get github.com/SamyRai/go-z-ai
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

// From the environment: ZAI_API_KEY, ZAI_API_BASE_URL, ZAI_REGION,
// ZAI_CHINA_API_KEY, ZAI_MONITOR_TIMEZONE.
c, err := client.NewClientFromEnv()

// Or explicitly:
c, err = client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
    // Optional: Region, BaseURL, Timeout, MaxRetries, RetryDelay, ChinaAPIKey,
    // UserAgent, Hooks
})
```

Services, all following `c.<Service>().<Method>(ctx, …)`:

| Accessor | Covers |
|---|---|
| `c.Chat()` | `Create`, `Stream` (iterator), `CreateAsync`, `RunWithTools` |
| `c.Anthropic()` | Anthropic-protocol `/v1/messages` (`Create`, `Stream`) |
| `c.Responses()` | OpenAI Responses protocol at `/api/v1` — the Codex surface (`Create`, `Stream`) |
| `c.Models()` | List, Get, text/vision/free filters |
| `c.Images()` / `c.Videos()` | Image (`Generate`, `GenerateAsync`), video (always async) |
| `c.Audio()` / `c.Voice()` | Transcription, TTS, voice cloning |
| `c.Layout()` / `c.FileParser()` | OCR + document-to-text for RAG |
| `c.Files()` / `c.Batch()` | Upload, batch jobs |
| `c.Agents()` | Z.AI specialized agents |
| `c.Embeddings()` / `c.Rerank()` / `c.Moderations()` | Retrieval + moderation |
| `c.Tools()` | WebSearch, WebReader, Tokenize |
| `c.Quota()` / `c.Account()` / `c.Detection()` | GLM Coding Plan quota and usage, account info, account-type detection |
| `c.GetAsyncResult()` / `c.WaitForResult()` | Shared polling for async tasks |

Things worth knowing:

- Defaults come from the catalog: `client.DefaultModel`,
  `client.DefaultFastModel`, `client.DefaultVisionModel`, and the OCR/ASR/TTS
  constants. `ChatRequest.ReasoningEffort` is validated against the model's
  accepted levels (`client.EffortLow`, `EffortHigh`, `EffortMax` for GLM-5.3);
  `client.CatalogEntry(model)` and `Pricing.Cost(usage)` expose context sizes
  and prices.
- `Chat().Stream`, `Anthropic().Stream`, and `Responses().Stream` return
  `iter.Seq2` iterators.
  Breaking out of the loop closes the stream, and an in-band error chunk ends it
  with an `*APIError`.
- Z.AI has no `json_schema` response format. For structured output, set
  `ResponseFormat: client.JSONObjectFormat()` and put
  `client.JSONSchemaPrompt(schema)` in the system prompt.
- `Config.Region` (`client.RegionGlobal` or `client.RegionChina`) owns every
  URL; `Config.BaseURL` defaults to `Region.PaaSBaseURL()`.

→ Full API with examples: **[Library Guide](docs/en/library-guide.md)**
→ Generated reference: [pkg.go.dev](https://pkg.go.dev/github.com/SamyRai/go-z-ai)

## Configuration

Credentials are resolved in this priority order (highest wins):

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
# ZAI_REGION=china                                   # if your key was issued on open.bigmodel.cn; selects every endpoint
# ZAI_CHINA_API_KEY=...                              # separate bigmodel.cn credential (embeddings/moderations)
# ZAI_MONITOR_TIMEZONE=UTC                           # zone of the quota/usage API (default UTC+8)
```

The library's `client.NewClientFromEnv()` reads the same variables from the
process environment (it does not load `.env`).

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
> placeholder values (`"your-zai-api-key-here"`) that illustrates the settings
> `go-z-ai coding load claude-code` writes. The program does not read it, it is
> not a real config, and it ships no credentials; run
> `go-z-ai coding load claude-code` for the current model mapping.
>
> ⚠️ **Usage policy.** Z.AI's coding endpoint is restricted to "officially
> supported tools" and prohibits SDK-based access; see
> [Coding Tools — Compliance](docs/en/coding-tools.md#compliance--usage-policy-).
> `go-z-ai` sends an identifying `User-Agent` header on every request and
> its `coding` subcommand wires officially-supported tools. Using `pkg/client`
> directly against the coding endpoint from a custom integration is at your
> own risk until explicit listing.

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
