# CLI Reference

Every command supports `--help` for its authoritative, always-up-to-date flag
list (`go-z-ai <command> --help`, `go-z-ai <command> <subcommand> --help`).
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
- [Responses endpoint (Codex protocol)](#responses-endpoint-codex-protocol)
- [Terminal UI](#terminal-ui)

## Global flags

These apply to every command:

| Flag | Description |
|---|---|
| `--api-key string` | Z.AI API key (or `ZAI_API_KEY` env var) |
| `--account string` | Use a stored account by name for this command (see [Accounts & Quota](accounts-and-quota.md)) |
| `--region string` | Regional gateway: `global` (api.z.ai, default) or `china` (open.bigmodel.cn). Aliases `cn`, `bigmodel`, `west`. Or `ZAI_REGION` env. Unknown values fall back to global. |
| `--base-url string` | Chat/PaaS API root (or `ZAI_API_BASE_URL` env). Default: the region's, e.g. `https://api.z.ai/api/paas/v4` |
| `--china-api-key string` | open.bigmodel.cn key for Embeddings/Moderations (or `ZAI_CHINA_API_KEY`; falls back to `--api-key`) |
| `--monitor-timezone string` | Timezone the quota/usage (monitor) API operates in (or `ZAI_MONITOR_TIMEZONE`; default CST/UTC+8). IANA names, `UTC`, or offsets like `+8` |
| `--config string` | Config file (default: `.env`) |
| `-v`, `--version` | Print the version and exit. Release builds (GoReleaser ldflags) print the tag; development builds print `dev` with commit and build date. |

Most result-producing commands take `--format text|json` (`models` commands
call the text mode `table`; `embeddings` and `moderations` default to `json`).
A few commands that only perform an action have no `--format`: `accounts
add|use|remove`, `coding auth|load|unload|doctor`, `coding mcp add|remove`,
`audio speech`, `files download`, and `validate`. Progress and status chatter
goes to stderr, so with `--format json` stdout stays valid JSON you can pipe
into `jq`.

`--region` (or `ZAI_REGION`) selects the gateway for every endpoint the CLI
talks to: chat and the other PaaS services, the Anthropic-compatible surface,
quota/usage, account (biz), agents, and account-type detection. Set it to
`china` when your key was issued on `open.bigmodel.cn`. `--base-url` overrides
only the chat/PaaS root and wins over the region's default there.
Embeddings and Moderations always use `open.bigmodel.cn`. When neither
`--region` nor `ZAI_REGION` is set, a stored account's region applies. See
[Accounts & Quota § Regional gateways](accounts-and-quota.md#regional-gateways-apizai--openbigmodelcn).

## Chat

```bash
go-z-ai chat create <message> [flags]
go-z-ai chat async-result <task-id>
```

`chat create` is the main entry point. Sampling settings (`--temperature`,
`--top-p`, `--max-tokens`) default to the model's own when left at `0`.

| Flag | Purpose |
|---|---|
| `--model string` | Default: the catalog's default chat model (`client.DefaultModel`, currently `glm-5.3`) |
| `--system string` | System message |
| `--stream` | Token-by-token streaming. With `--format json`, one chunk per line |
| `--async` | Submit without waiting; poll with `chat async-result <task-id>` |
| `--temperature float`, `--top-p float`, `--max-tokens int` | Sampling controls (`0` = model default) |
| `--do-sample` | Sample (default `true`); `--do-sample=false` is greedy decoding |
| `--stop strings` | Stop sequence (the API honors one) |
| `--thinking string` | `enabled` or `disabled` (GLM-5.3 models always reason) |
| `--effort string` | Reasoning effort: `max\|xhigh\|high\|medium\|low\|minimal\|none`. Validated against the model's catalog entry: GLM-5.3 accepts `low`, `high`, `max`; GLM-5.2 accepts every level |
| `--show-reasoning` | Print the reasoning (to stderr in text mode) |
| `--json` | Ask for a JSON object response |
| `--json-schema string` | JSON object response matching a schema: `@file.json` or inline JSON. Z.AI has no `json_schema` response format, so the schema is added to the system message |
| `--tool string` | Function-calling tool declarations: `@tools.json` or inline JSON array |
| `--tool-stream` | Stream tool-call arguments incrementally (GLM-4.6+) |
| `--image string`, `--video string`, `--file string` (repeatable) | Attach an image, video, or document: a URL, or `@path` to a local file (base64-encoded) |
| `--format text\|json` | Output format |

```bash
go-z-ai chat create "Summarize this in 3 bullets" --stream
go-z-ai chat create "Plan a migration" --effort max --show-reasoning
go-z-ai chat create "Extract fields" --json-schema @schema.json --format json
go-z-ai chat create "Describe this" --image @photo.jpg --model glm-5.3-flash
```

Attachments need a model with the matching capability (`video`, `file`,
`vision` in `go-z-ai models list`). The default chat model is text-only in the
catalog; `glm-5.3-flash` (`client.DefaultVisionModel`) is natively multimodal
(image, video, file input).

Tool calls are printed, not executed, by the CLI — see
[Library Guide § Function calling](library-guide.md#function-calling) for the
Go `RunWithTools` auto-executing loop.

`chat async-result`, `image status`, and `video status` share one handler:
in text mode the task status goes to stderr and the result (message, image
URLs, or video URLs) to stdout; `--format json` prints the full task result.

> **Vision + tool-calling can return HTTP 401.** Community reports (e.g.
> [claude-code-router#1491](https://github.com/musistudio/claude-code-router/issues/1491))
> show that combining a vision model (`--image` on `glm-4.6v`/`glm-4.5v`) with
> function-calling tools (`--tool`) in the same request is rejected with a 401
> on some GLM configurations — an authenticated key still fails only for that
> combination. If you hit this, split the work: use a vision model for the
> image turn and a text model for the tool-calling turn, rather than sending
> images and tools together. Not reproduced against a live account here.

## Models

```bash
go-z-ai models list
go-z-ai models get <model-id>
go-z-ai models text | vision | free
```

The `/models` endpoint returns bare IDs; context size, max output, prices
(USD per 1M tokens), capabilities (`text`, `vision`, `video`, `file`,
`thinking`, `tools`, `code`, `ocr`, `audio`), and accepted reasoning efforts
come from the curated catalog in `pkg/client/models_catalog.go`. Models the
catalog does not know still appear, with `-` for unknown values. `models get`
also shows the effort levels.

## Accounts, usage, and quota

Covered in depth in [Accounts & Quota](accounts-and-quota.md). Quick reference:

```bash
go-z-ai accounts add <name> --api-key <key> [--type coding_plan|pay_as_you_go] [--region global|china] [--base-url-override URL] [--force]
go-z-ai accounts list [--format json] [--reveal]   # keys masked by default; --reveal for export
go-z-ai accounts use <name>
go-z-ai accounts show [name] [--format json] [--reveal]
go-z-ai accounts current                            # shorthand for 'accounts show' (active account)
go-z-ai accounts quota [--only name...]
go-z-ai accounts usage [--days N] [--today] [--metric model|tool|both] [--only name...]
go-z-ai accounts remove <name> [--yes]

go-z-ai account detect                              # type and region of the current key (free)
go-z-ai account status [--watch 5m]                 # can the key authenticate and spend?
go-z-ai account balance                             # pay-as-you-go wallet balance
go-z-ai account subscriptions                       # GLM Coding Plan subscriptions

go-z-ai usage quota                                 # Coding Plan quota windows, reset times, pace
go-z-ai validate                                    # free request (list models) to check the key
```

- `accounts add` detects the account type and region with a free probe unless
  `--type` is given. `--region` says which gateway the key was issued on; with
  detection it is where the probe starts. With `--type` nothing is detected,
  so pass `--region china` for a China key.
- `accounts quota` and `accounts usage` cover all stored accounts (limit with
  `--only`, repeatable); pay-as-you-go accounts are skipped because the
  monitor endpoints are coding-plan only. `accounts usage` buckets hourly for
  8 days or fewer and daily for 9 or more (`--days` defaults to 14).
- `account status` checks a Coding Plan key for free through the quota
  endpoint. A pay-as-you-go key needs one minimal billed request (Z.AI has no
  balance-check API for them), and `--watch` bills it on every re-check.
- `account balance` and `account subscriptions` use the biz endpoints, which
  are not yet verified against a live account (see [Roadmap](roadmap.md)).
- `usage quota` shows each credit window with its reset time and pace, plus a
  peak-hours notice (Mon–Fri 14:00–18:00 UTC+8). `usage` has no other
  subcommands; for several accounts at once use `accounts quota` and
  `accounts usage`.

## Coding tools (GLM Coding Plan)

Wires Claude Code, Codex, OpenCode, Crush, or Factory Droid to use your GLM
Coding Plan. Full walkthrough: [Coding Tools](coding-tools.md).

```bash
go-z-ai coding auth <plan> <key> [--no-validate]   # validate + store a credential
go-z-ai coding auth revoke
go-z-ai coding auth reload <tool>     # re-push stored creds into a tool's config
go-z-ai coding load <tool> [--plan ...] [--key ...]   # write it into a tool's config
go-z-ai coding unload <tool>          # says so and succeeds if the tool isn't configured
go-z-ai coding status [--format json]
go-z-ai coding tools [--format json]  # list supported tools + install status
go-z-ai coding doctor                 # health check; exits non-zero on problems

go-z-ai coding mcp add <tool> [--server id]... [--plan ...] [--key ...]
go-z-ai coding mcp remove <tool> [--server id]...
go-z-ai coding mcp status [--format json]
```

Tool IDs: `claude-code`, `codex`, `opencode`, `crush`, `factory-droid`
(aliases `claude`, `droid`, `factory`). Plans: `glm_coding_plan_global`,
`glm_coding_plan_china`.

`coding load` and `coding auth` accept Claude Code tuning flags: `--haiku`,
`--sonnet`, `--opus` (override a tier's model), `--no-model-mapping`,
`--auto-compact-window`, `--max-thinking-tokens`, `--max-output-tokens`.

`coding mcp` registers Z.AI's four official MCP servers; `--server` (repeatable)
selects some of them, and the default is every server the tool supports.
Codex takes only the local Vision server: its streamable-HTTP MCP client
rejects the hosted ones (openai/codex#14793), so asking for one fails. Calls
to the servers draw plan quota.

| Server ID | What it does |
|---|---|
| `zai-mcp-server` | Vision: screenshots, diagrams, UI, images and video. Runs locally via `npx`, so it needs Node.js |
| `web-search-prime` | Web search (hosted) |
| `web-reader` | Fetch and read web pages (hosted) |
| `zread` | Read and search GitHub repositories (hosted) |

Hosted servers authenticate with the plan key against the plan's region.

## Files & batch

```bash
go-z-ai files upload <file> [--purpose batch|user_data|agent|file-extract|code-interpreter|voice-clone-input]
go-z-ai files list [--purpose ...]
go-z-ai files delete <file-id>
go-z-ai files download <file-id> <output-path>

go-z-ai batch create <input-file-id> [--endpoint ...] [--auto-delete-input] [--metadata key=value]...
go-z-ai batch status <batch-id>
go-z-ai batch list [--after ...] [--limit N]
go-z-ai batch cancel <batch-id>
```

`files upload` defaults to `--purpose batch`. Batch jobs process many
chat-completion or embedding requests from a JSONL file asynchronously —
upload it first, then create the batch with the resulting file ID.
`--auto-delete-input` deletes the input file when the batch finishes;
`--metadata` is repeatable. `batch status` prints the output and error file
IDs, which you fetch with `files download`.

## Media generation

```bash
# Images — default model glm-image (cogview-4-250304 also supported)
go-z-ai image generate <prompt> [--model glm-image|cogview-4-250304] [--size ...] [--quality hd|standard] [--async]
go-z-ai image status <id>
# --quality: hd is the default (~20s); standard is faster (~5-10s).
# --async is supported by glm-image only.

# Video — always async (cogvideox-3 | viduq1-text | viduq1-image | viduq1-start-end | vidu2-image | vidu2-start-end | vidu2-reference)
go-z-ai video generate --prompt "..." [--model ...] [--image URL]... [--size ...] [--aspect-ratio ...] \
    [--duration N] [--audio] [--off-peak]
go-z-ai video status <id>
# --off-peak queues the task for off-peak processing at a lower price.
# Model-specific: --fps and --quality (cogvideox-3), --style (viduq1-text),
# --movement (Vidu), --aspect-ratio (Vidu text/reference). Default model cogvideox-3.

# Audio
go-z-ai audio transcribe <file> [--model ...] [--hotword word]... [--prompt ...]   # .wav/.mp3, <=25MB, <=30s
go-z-ai audio speech <text> <output-path> [--voice ...] [--speed N] [--audio-format wav|pcm]

# Voice cloning (pairs with audio speech --voice)
go-z-ai voice clone <voice-name> <sample-file-id> <preview-text>
go-z-ai voice list [--name ...] [--type OFFICIAL|PRIVATE]
go-z-ai voice delete <voice-id>
```

`audio transcribe` defaults to the catalog's ASR model (`client.DefaultASRModel`)
and `audio speech` uses the TTS model (`client.DefaultTTSModel`). `--voice`
takes a system voice (`tongtong`, the default, `chuichui`, `xiaochen`, `jam`,
`kazi`, `douji`, `luodo`) or a cloned voice ID; `--speed` is 0.5–2. The sample
file for `voice clone` must already be uploaded with
`files upload --purpose voice-clone-input`.

## Document parsing & OCR

```bash
# Layout OCR (glm-ocr) — image/PDF into Markdown
go-z-ai ocr parse <file-or-url> [--start-page N] [--end-page N]
go-z-ai ocr handwriting <file> [--probability] [--language ...]

# Document parser (RAG/retrieval preprocessing) — a separate product from OCR
go-z-ai parser parse <file> <file-type>              # synchronous
go-z-ai parser create <file> <tool-type> <file-type> # async: lite|expert|prime
go-z-ai parser result <task-id> <format>              # text|download_link
```

`parser` and `ocr` solve different problems: OCR extracts layout/text from
images; the parser is built for turning documents into RAG-ready text and
accepts more tool tiers.

## Retrieval helpers

```bash
go-z-ai embeddings create <text> [--model embedding-3|embedding-2] [--dimensions N]
go-z-ai rerank <query> <documents...> [--top-n N]
```

Embeddings route to `open.bigmodel.cn` — see
[Accounts & Quota § Regional gateways](accounts-and-quota.md#regional-gateways-apizai--openbigmodelcn)
for why, and what that means for authentication. `--dimensions` applies to
`embedding-3` only (256, 512, 1024, or 2048). Rerank uses the chat/PaaS root
(`--base-url`, or the `--region` default); it is not pinned to the China host.

## Content moderation

```bash
go-z-ai moderations check <text>
```

Routes to `open.bigmodel.cn` — same note as Embeddings above.

## Agents

```bash
go-z-ai agents invoke <agent-id> <message> [--source-lang ...] [--target-lang ...] [--var key=value]...
go-z-ai agents async-result <agent-id> <async-id> [--conversation-id ID] [--var key=value]...
```

Invokes Z.AI's specialized agents (translation, slide/poster generation, video
effect templates). `--var` sets an agent custom variable and is repeatable;
`--source-lang` and `--target-lang` are shorthands for the `source_lang` and
`target_lang` variables. `invoke` prints the conversation ID to stderr; pass
it back to `async-result` with `--conversation-id`. Note: the Agents API
returns HTTP 200 even when an invocation fails at the business level (e.g.
insufficient balance) — the CLI reports that failure from the response body as
a command error.

## Tools (web search, reader, tokenizer)

```bash
go-z-ai tools web-search <query> [--engine ...] [--count N] [--recency ...] [--domain ...] [--content-size medium|high]
go-z-ai tools web-reader <url> [--no-images]
go-z-ai tools tokenizer <text> [--model ...]
```

`web-search` flags:

| Flag | Purpose |
|---|---|
| `--engine string` | `search-prime` (default), `search_std`, `search_pro`, `search_pro_sogou`, `search_pro_quark` |
| `--count int` | Number of results, 1–50 (default 10) |
| `--recency string` | `oneDay`, `oneWeek`, `oneMonth`, `oneYear`, `noLimit` |
| `--domain string` | Restrict results to one domain |
| `--content-size string` | Result content size: `medium` or `high` |

`tokenizer` counts tokens for a single user message; `--model` defaults to the
catalog's default chat model.

## Anthropic-compatible endpoint

```bash
go-z-ai anthropic messages <prompt> [--model ...] [--max-tokens 1024] \
    [--system ...] [--temperature ...] [--thinking-budget N] [--stream]
```

Calls Z.AI's Anthropic-protocol surface (`/api/anthropic/v1/messages` on the
selected `--region`) — the same endpoint the GLM Coding Plan points Claude Code
at — instead of the OpenAI-style `chat create`. `--model` defaults to the
catalog's default chat model; `--max-tokens` is required by the Messages API
and defaults to 1024. Prints the message text (or streams text deltas with
`--stream`; with `--format json`, one event per line); `--thinking-budget N`
enables extended thinking and prints the reasoning to stderr. See the
[Library Guide](library-guide.md#anthropic-compatible-messages-api) for the
Go API.

## Responses endpoint (Codex protocol)

```bash
go-z-ai responses create <prompt> [--model ...] [--instructions ...] \
    [--effort low|high|max] [--max-output-tokens N] [--stream] [--show-reasoning]
```

Calls Z.AI's OpenAI Responses-protocol endpoint (`/api/v1/responses` on the
selected `--region`) — the surface Codex is configured against, documented
for the GLM Coding Plan. `--model` defaults to the catalog's default chat
model, and `--effort` is validated against it. Prints the output text (tool
calls and, with `--show-reasoning`, reasoning go to stderr); `--stream`
prints text deltas, or one event per line with `--format json`. See the
[Library Guide](library-guide.md) for the Go API.

## Terminal UI

```bash
go-z-ai tui
```

Launches a full-screen terminal UI with Chat, Models, Usage, Accounts, Coding,
Media, and Tools tabs — the same functionality as the CLI commands above, in
one interactive session. Press `?` or `F1` for the key bindings.
