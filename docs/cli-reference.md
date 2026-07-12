# CLI Reference

Every command supports `--help` for its authoritative, always-up-to-date flag
list (`zai-client <command> --help`, `zai-client <command> <subcommand> --help`).
This page is the organized tour; treat `--help` as the source of truth if the
two ever disagree.

## Global flags

These apply to every command:

| Flag | Description |
|---|---|
| `--api-key string` | Z.AI API key (or `ZAI_API_KEY` env var) |
| `--base-url string` | API base URL (default: `https://api.z.ai/api/paas/v4`) |
| `--account string` | Use a stored account by name for this command (see [Accounts & Quota](accounts-and-quota.md)) |
| `--china-api-key string` | open.bigmodel.cn key for Embeddings/Moderations (or `ZAI_CHINA_API_KEY`; falls back to `--api-key`) |
| `--config string` | Config file (default: `.env`) |

Most output-producing commands also take `--format table\|json`.

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
| `--thinking string`, `--effort string` | Deep-thinking mode and effort level (`max\|high\|medium\|low\|minimal\|none`) |
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
zai-client accounts list
zai-client accounts use <name>
zai-client accounts show [name]
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
# Images (glm-image | cogview-4-250304)
zai-client image generate <prompt> [--model ...] [--size ...] [--quality hd|standard] [--async]
zai-client image status <id>

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
[Accounts & Quota § China platform key](accounts-and-quota.md#china-platform-key)
for why, and what that means for authentication.

## Content moderation

```bash
zai-client moderations check <text>
```

Also routes to `open.bigmodel.cn` — same note as Embeddings above.

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

## Terminal UI

```bash
zai-client tui
```

Launches a full-screen terminal UI with Chat, Models, Usage, Accounts, Coding,
Media, and Tools tabs — the same functionality as the CLI commands above, in
one interactive session.
