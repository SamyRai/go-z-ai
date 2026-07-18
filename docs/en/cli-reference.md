# CLI Reference

Every command supports `--help` for its authoritative, always-up-to-date flag
list (`zai-client <command> --help`, `zai-client <command> <subcommand> --help`).
This page is the organized tour; treat `--help` as the source of truth if the
two ever disagree.

## Contents

- [Global flags](#global-flags)
- [Chat](#chat)
- [Models](#models)
- [Accounts, usage, and quota](#accounts-usage-and-quota)
- [Coding tools (GLM Coding Plan)](#coding-tools-glm-coding-plan)
- [Files & batch](#files--batch)
- [Media generation](#media-generation)
- [Document parsing & OCR](#document-parsing--ocr)
- [Retrieval helpers](#retrieval-helpers)
- [Content moderation](#content-moderation)
- [Agents](#agents)
- [Tools (web search, reader, tokenizer)](#tools-web-search-reader-tokenizer)
- [Anthropic-compatible endpoint](#anthropic-compatible-endpoint)
- [Terminal UI](#terminal-ui)

## Global flags

These apply to every command:

| Flag | Description |
|---|---|
| `--api-key string` | Z.AI API key (or `ZAI_API_KEY` env var) |
| `--base-url string` | API base URL (default: `https://api.z.ai/api/paas/v4`) |
| `--account string` | Use a stored account by name for this command (see [Accounts & Quota](accounts-and-quota.md)) |
| `--china-api-key string` | open.bigmodel.cn key for Embeddings/Moderations (or `ZAI_CHINA_API_KEY`; falls back to `--api-key`) |
| `--region string` | Regional gateway for monitor/biz/agents/detection: `global` (api.z.ai, default) or `china` (open.bigmodel.cn). Aliases `cn`, `bigmodel`, `west`. Or `ZAI_REGION` env. Does not override `--base-url`. Unknown values fall back to global. |
| `--config string` | Config file (default: `.env`) |
| `--version` | Print version (tag, commit, build date) and exit. Populated by GoReleaser ldflags in release builds; `dev` otherwise. |

Every result-producing command takes `--format text\|json` (a few default to
`json` where the payload is machine-oriented — e.g. `embeddings`,
`moderations`). In `json` mode, progress/status chatter goes to stderr so
stdout stays valid JSON you can pipe into `jq`.

Set `--region china` (or `ZAI_REGION=china`) when your key was issued on
`open.bigmodel.cn`, so quota/usage, account-info, agents, and account-type
detection land on the matching host. It does **not** change the chat base URL
(use `--base-url` for that) or the Embeddings/Moderations host (always the
China platform). See [Accounts & Quota § Regional gateways](accounts-and-quota.md#regional-gateways-apiza--openbigmodelcn).

## Chat

```bash
zai-client chat create [message] [flags]
zai-client chat simple [model] [message]
zai-client chat async-result [task-id]
```

`chat create` is the main entry point:

| Flag | Purpose |
|---|---|
| `--model string` | Default `glm-5.2` |
| `--stream` | Token-by-token streaming |
| `--async` | Submit without waiting; poll with `chat async-result <task-id>` |
| `--temperature float`, `--top-p float`, `--max-tokens int` | Sampling controls |
| `--system string` | System message |
| `--thinking string`, `--effort string` | Deep-thinking mode and effort level (`max\|xhigh\|high\|medium\|low\|minimal\|none`; `xhigh`→`max` is GLM-5.2 only) |
| `--show-reasoning` | Print `reasoning_content` to stderr |
| `--json-schema string` | Structured output: `@file.json` or inline JSON |
| `--tool string` | Function-calling tool declarations: `@tools.json` or inline JSON array |
| `--image string` (repeatable) | Attach an image: a URL, or `@path` to a local file (base64-encoded). Requires a vision model (`glm-4.6v`/`glm-4.5v`) |
| `--stop strings` | Stop sequences (repeatable, max 4) |
| `--format text\|json` | Output format |

```bash
zai-client chat create "Summarize this in 3 bullets" --model glm-5.2 --stream
zai-client chat create "Describe this" --image @photo.jpg --model glm-4.6v
zai-client chat create "Extract fields" --json-schema @schema.json
```

Tool calls are printed, not executed, by the CLI — see
[Library Guide § Function calling](library-guide.md#function-calling) for the
Go `RunWithTools` auto-executing loop.

> **Vision + tool-calling can return HTTP 401.** Community reports (e.g.
> [claude-code-router#1491](https://github.com/musistudio/claude-code-router/issues/1491))
> show that combining a vision model (`--image` on `glm-4.6v`/`glm-4.5v`) with
> function-calling tools (`--tool`) in the same request is rejected with a 401
> on some GLM configurations — an authenticated key still fails only for that
> combination. If you hit this, split the work: use a vision model for the
> image turn and a text model (`glm-5.2`) for the tool-calling turn, rather
> than sending images and tools together. Not yet reproduced against a live
> account here — see [Roadmap](roadmap.md).

## Models

```bash
zai-client models list [--pricing]
zai-client models get [model-id]
zai-client models text | vision | free
```

## Accounts, usage, and quota

Covered in depth in [Accounts & Quota](accounts-and-quota.md). Quick reference:

```bash
zai-client accounts add <name> --api-key <key> [--type coding_plan|pay_as_you_go]
zai-client accounts list [--format json] [--reveal]   # keys masked by default; --reveal for export
zai-client accounts use <name>
zai-client accounts show [name] [--format json] [--reveal]
zai-client accounts current                            # shorthand for 'accounts show' (active account)
zai-client accounts quota [--only name...]
zai-client accounts usage [--days N] [--today] [--metric model|tool|both]
zai-client accounts remove <name> [--yes]

zai-client usage quota | summary | account | billing | check [--watch] | detect
zai-client account info | status
zai-client validate
```

## Coding tools (GLM Coding Plan)

Wires Claude Code, OpenCode, Crush, Factory Droid, or Cursor to use your GLM
Coding Plan. Full walkthrough: [Coding Tools](coding-tools.md).

```bash
zai-client coding auth <plan> <key>      # validate + store a credential
zai-client coding auth revoke
zai-client coding auth reload <tool>     # re-push stored creds into a tool's config
zai-client coding load <tool>            # write it into a tool's config
zai-client coding unload <tool>
zai-client coding status
zai-client coding tools                  # list supported tools + install status
zai-client coding doctor                 # health check

zai-client coding mcp add <tool>         # register Z.AI's Vision MCP server
zai-client coding mcp status
zai-client coding mcp remove <tool>
```

## Files & batch

```bash
zai-client files upload <file> [--purpose batch|code-interpreter|agent|voice-clone-input]
zai-client files list [--purpose ...]
zai-client files delete <file-id>
zai-client files download <file-id> <output-path>

zai-client batch create <input-file-id> [--endpoint ...]
zai-client batch status <batch-id>
zai-client batch list [--after ...] [--limit N]
zai-client batch cancel <batch-id>
```

Batch jobs process many chat-completion requests from a JSONL file
asynchronously — upload it first, then create the batch with the resulting
file ID.

## Media generation

```bash
# Images — default model glm-image (cogview-4-250304 also supported)
zai-client image generate <prompt> [--model glm-image|cogview-4-250304] [--size ...] [--quality hd|standard] [--async]
zai-client image status <id>
# --quality: hd is the default (~20s); standard is faster (~5-10s).

# Video — always async (cogvideox-3 | viduq1-text | viduq1-image | vidu2-image | ...)
zai-client video generate --prompt "..." [--model ...] [--duration N] [--aspect-ratio ...]
zai-client video status <id>

# Audio
zai-client audio transcribe <file>                       # glm-asr, .wav/.mp3, <=25MB, <=30s
zai-client audio speech <text> <output-path> [--voice ...] [--speed N] [--format wav|pcm]

# Voice cloning (pairs with audio speech --voice)
zai-client voice clone <voice-name> <sample-file-id> <preview-text>
zai-client voice list [--name ...] [--type OFFICIAL|PRIVATE]
zai-client voice delete <voice-id>
```

## Document parsing & OCR

```bash
# Layout OCR — image/PDF into Markdown
zai-client ocr parse <file-or-url> [--start-page N] [--end-page N]
zai-client ocr handwriting <file> [--probability]

# Document parser (RAG/retrieval preprocessing) — a separate product from OCR
zai-client parser parse <file> <file-type>              # synchronous
zai-client parser create <file> <tool-type> <file-type> # async: lite|expert|prime
zai-client parser result <task-id> <format>              # text|download_link
```

`parser` and `ocr` solve different problems: OCR extracts layout/text from
images; the parser is built for turning documents into RAG-ready text and
accepts more tool tiers.

## Retrieval helpers

```bash
zai-client embeddings create <text> [--model embedding-3|embedding-2] [--dimensions N]
zai-client rerank <query> <documents...> [--top-n N]
```

Embeddings route to `open.bigmodel.cn` — see
[Accounts & Quota § Regional gateways](accounts-and-quota.md#regional-gateways-apiza--openbigmodelcn)
for why, and what that means for authentication. Rerank uses the default
`--base-url` (it is not pinned to the China host).

## Content moderation

```bash
zai-client moderations check <text>
```

Routes to `open.bigmodel.cn` — same note as Embeddings above.

## Agents

```bash
zai-client agents invoke <agent-id> <message> [--source-lang ...] [--target-lang ...]
zai-client agents async-result <agent-id> <async-id>
```

Invokes Z.AI's specialized agents (translation, slide/poster generation, video
effect templates). Note: the Agents API returns HTTP 200 even when an
invocation fails at the business level (e.g. insufficient balance) — the CLI
reports that failure from the response body, not as a command error.

## Tools (web search, reader, tokenizer)

```bash
zai-client tools web-search <query> [--engine ...] [--count N]
zai-client tools web-reader <url> [--no-images]
zai-client tools tokenizer <text> [--model ...]
```

## Anthropic-compatible endpoint

```bash
zai-client anthropic messages <prompt> [--model glm-4.6] [--max-tokens 1024] \
    [--system ...] [--temperature ...] [--thinking-budget N] [--stream]
```

Calls Z.AI's Anthropic-protocol surface (`/api/anthropic/v1/messages`) — the
same endpoint the GLM Coding Plan points Claude Code at — instead of the
OpenAI-style `chat create`. Prints the message text (or streams text deltas
with `--stream`); `--thinking-budget N` enables extended thinking and prints
the reasoning to stderr. See the
[Library Guide](library-guide.md#anthropic-compatible-messages-api) for the
Go API.

## Terminal UI

```bash
zai-client tui
```

Launches a full-screen terminal UI with Chat, Models, Usage, Accounts, Coding,
Media, and Tools tabs — the same functionality as the CLI commands above, in
one interactive session.
