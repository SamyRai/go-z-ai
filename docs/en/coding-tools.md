# Coding Tools (GLM Coding Plan)

`go-z-ai coding` configures third-party coding assistants to use your GLM
Coding Plan instead of their default provider. It's a Go port of Z.AI's
official `@z_ai/coding-helper` ("chelper") CLI
([docs](https://docs.z.ai/devpack/extension/coding-tool-helper)), sharing the
same credential file so the two tools can be used interchangeably.

## Supported tools

| Tool | ID (aliases) | Config file it writes |
|---|---|---|
| Claude Code | `claude-code` (`claude`) | `~/.claude/settings.json` (+ `~/.claude.json`: onboarding flag and MCP servers) |
| Codex | `codex` | `~/.codex/config.toml` (+ `~/.codex/models.json`: model metadata) |
| OpenCode | `opencode` | `~/.config/opencode/opencode.json` |
| Crush | `crush` | `~/.config/crush/crush.json` |
| Factory Droid | `factory-droid` (`droid`, `factory`) | `~/.factory/settings.json` (MCP servers in `~/.factory/mcp.json`) |

Paths are under your home directory. Run `go-z-ai coding tools` to see install
status (is the tool's binary on `PATH`: `claude`, `codex`, `opencode`,
`crush`, `droid`) and the exact resolved paths on your machine.

Cursor is not supported (earlier releases wrote a Cursor settings file).
Z.AI documents Cursor as a GUI-only setup: OpenAI protocol, your key, and the
plan's coding endpoint as the base URL
([docs](https://docs.z.ai/devpack/tool/cursor)).

### What each tool's config gets

`coding load <tool>` merges the plan into the tool's own config and leaves
unrelated settings alone.

- **Claude Code**: in `~/.claude/settings.json`, an `env` block with the key,
  the plan's Anthropic endpoint, the tier mapping and the other variables
  listed under [Claude Code: model mapping](#claude-code-model-mapping). In
  `~/.claude.json` it sets `hasCompletedOnboarding: true` unless that is
  already set. It also removes a stale `ANTHROPIC_API_KEY` from the `env`
  block. See [Claude Code's docs](https://docs.z.ai/devpack/tool/claude) for
  the manual equivalent.
- **Codex**: in `~/.codex/config.toml`, `model_provider = "ZAI"`,
  `model = <client.DefaultModel>`, `model_reasoning_effort` (the strongest
  level the model takes, `max` for GLM-5.3), `model_catalog_json =
  "~/.codex/models.json"`, and a `[model_providers.ZAI]` table with the plan's
  Responses endpoint as `base_url`, the key as `experimental_bearer_token`,
  and `wire_api = "responses"`. In `~/.codex/models.json` it writes the
  model's metadata from the catalog (reasoning levels, context window, input
  modalities), replacing an older entry for the same model and keeping the
  others. This is the format of Z.AI's docs and the official helper.
  ([docs](https://docs.z.ai/devpack/tool/codex))
- **OpenCode**: a provider entry named after OpenCode's built-in plan
  provider, `zai-coding-plan` (Global plan) or `zhipuai-coding-plan` (China
  plan), holding `options.apiKey`; the other plan's entry is removed. It sets
  `$schema`, and sets `model` and `small_model` to
  `<provider>/<client.DefaultModel>` and `<provider>/<client.DefaultFastModel>`
  only when they are unset or already point at a plan provider, so your own
  choice survives. ([docs](https://docs.z.ai/devpack/tool/opencode))
- **Crush**: `providers.zai` with `id: "zai"`, `name: "ZAI Provider"`, the
  plan's coding endpoint as `base_url`, and `api_key`. No model is written;
  pick one in Crush's UI. ([docs](https://docs.z.ai/devpack/tool/crush))
- **Factory Droid**: two `customModels` entries for `client.DefaultModel`, one
  over the Anthropic protocol (`provider: "anthropic"`, the plan's Anthropic
  endpoint) and one over the OpenAI protocol (`provider:
  "generic-chat-completion-api"`, the coding endpoint). `maxOutputTokens` comes
  from the model catalog. Display names look like `GLM-5.3 [GLM Coding Plan
  Global] - Anthropic`; any existing entry whose display name contains
  `GLM Coding Plan` is replaced (this also covers entries written by the
  official helper), other entries are kept.
  ([docs](https://docs.z.ai/devpack/tool/droid))

How the files are written:

- The API key goes into each tool's config in plain text, as those tools
  expect. Files are written atomically with `0600` permissions.
- Before a config file is overwritten for the first time, a copy is kept next
  to it as `<file>.zai.bak`. An existing backup is never replaced. Symlinked
  configs are written through, so dotfile managers keep working.
- Configs are read and rewritten as JSON (indented, keys sorted), or as
  TOML for a `.toml` file (Codex). A TOML rewrite drops comments, as the
  official helper's does; the `.zai.bak` backup keeps the original. A config
  that can't be parsed makes the command fail with a parse error; it is not
  overwritten.
- `coding load` for a tool that is not installed still writes its config.

## Plans

| Plan identifier | Gateway | OpenAI-compatible endpoint | Anthropic endpoint | Responses endpoint (Codex) |
|---|---|---|---|---|
| `glm_coding_plan_global` | `https://api.z.ai` | `https://api.z.ai/api/coding/paas/v4` | `https://api.z.ai/api/anthropic` | `https://api.z.ai/api/v1` |
| `glm_coding_plan_china` | `https://open.bigmodel.cn` | `https://open.bigmodel.cn/api/coding/paas/v4` | `https://open.bigmodel.cn/api/anthropic` | `https://open.bigmodel.cn/api/v1` |

Pick whichever matches where your GLM Coding Plan subscription lives. The plan
decides the region of everything `coding` writes (tool endpoints, hosted MCP
URLs, the Vision server's mode). All URLs come from `client.Region` in
`pkg/client/region.go`. The global `--region`, `--api-key` and `--account`
flags do not apply to `coding`; use the plan and `--key`/`--plan` instead.

## Quickstart

```bash
# 1. Store and validate your GLM Coding Plan key (one-time)
go-z-ai coding auth glm_coding_plan_global YOUR_KEY

# 2. Load it into a tool
go-z-ai coding load claude-code
# tool IDs: claude-code, codex, opencode, crush, factory-droid
# aliases also work: claude, droid, factory

# 3. Check everything's wired up
go-z-ai coding status
go-z-ai coding doctor
```

Credentials live at `~/.chelper/config.yaml` (byte-compatible with the
official Node helper; keys `lang`, `plan`, `api_key`) — `coding auth` writes
there once, and `coding load` reads from it for every tool unless you pass
`--key`/`--plan` overrides.

To stop using Z.AI for a tool without losing your stored credential:

```bash
go-z-ai coding unload claude-code
```

This removes only the Z.AI-specific fields `load` added (Claude Code's managed
`env` variables, Codex's `ZAI` provider and — while it is still the active
provider — its top-level model settings, OpenCode's plan provider and plan
`model`/`small_model`, Crush's `providers.zai`, Droid's plan `customModels`
entries); it does not touch the rest of your existing config file. Codex's
`models.json` stays, as the official helper leaves it. It does not remove MCP servers
(use `coding mcp remove`), and Claude Code's `hasCompletedOnboarding` stays.

If the tool's config holds no Z.AI plan, `unload` says
`<Tool> is not configured for a Z.AI plan; nothing to remove.` and exits 0, so
it is safe to run repeatedly in scripts. A tool counts as configured only
when its config points at a plan endpoint (Claude Code's `ANTHROPIC_BASE_URL`,
Codex's `model_providers.ZAI.base_url`, Crush's `providers.zai.base_url`) or
holds a plan entry (OpenCode, Droid); a key set for another provider is left
alone.

The TUI's coding tab (`go-z-ai tui`) offers the same actions: `a` auth, `l`
load, `u` unload, `m` register the MCP servers the tool supports, `r` rescan. It loads with the
default Claude Code tuning below.

## Claude Code: model mapping

`load` writes the variables Z.AI documents for Claude Code, including a mapping
of Claude Code's model tiers to GLM models through `ANTHROPIC_DEFAULT_*_MODEL`
([docs](https://docs.z.ai/devpack/tool/claude)):

| Env var | Value |
|---|---|
| `ANTHROPIC_AUTH_TOKEN` | your key (sent as `Authorization: Bearer`, which Z.AI's Anthropic endpoint expects; `ANTHROPIC_API_KEY` is not used) |
| `ANTHROPIC_BASE_URL` | the plan's Anthropic endpoint |
| `API_TIMEOUT_MS` | `3000000` |
| `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC` | `1` |
| `ANTHROPIC_DEFAULT_HAIKU_MODEL` | `client.DefaultFastModel` + `[1m]` (`glm-5.3-flash[1m]`) |
| `ANTHROPIC_DEFAULT_SONNET_MODEL` | `client.DefaultModel` + `[1m]` (`glm-5.3[1m]`) |
| `ANTHROPIC_DEFAULT_OPUS_MODEL` | `client.DefaultModel` + `[1m]` (`glm-5.3[1m]`) |
| `CLAUDE_CODE_AUTO_COMPACT_WINDOW` | the main model's context size, 1048576 (set it to your own value with the flag below) |
| `MAX_THINKING_TOKENS`, `CLAUDE_CODE_MAX_OUTPUT_TOKENS` | only when you pass the matching flag |

The model IDs come from `pkg/client/models_catalog.go`, so a catalog refresh
updates what `load` writes. The `[1m]` suffix is a Claude Code convention that
opts a model into its 1M-token context; it belongs only in these variables,
never in an API request. `load` adds it to every model whose catalog context
is 1M tokens or more, which covers both defaults, as the official helper
does. Values you pass with `--haiku`/`--sonnet`/`--opus` are written verbatim,
so add `[1m]` yourself for a 1M-context model.

Z.AI's pages are not fully consistent on the mapping:
`docs.z.ai/devpack/tool/claude` splits it (flash for haiku, flagship for
sonnet and opus, as above), while `docs.z.ai/devpack/latest-model` shows the
flash model for all three tiers. The official helper (0.1.1) also writes the
flash model for all three. Override any tier, or opt out entirely:

```bash
# Use the flash model for every tier
go-z-ai coding load claude-code \
  --sonnet 'glm-5.3-flash[1m]' --opus 'glm-5.3-flash[1m]'

# Leave Claude Code's model selection alone
go-z-ai coding load claude-code --no-model-mapping
```

Tuning flags, all optional:

| Flag | Effect |
|---|---|
| `--haiku`, `--sonnet`, `--opus` | Override that tier's model ID |
| `--no-model-mapping` | Omit the three `ANTHROPIC_DEFAULT_*_MODEL` variables |
| `--auto-compact-window int` | Set `CLAUDE_CODE_AUTO_COMPACT_WINDOW`; defaults to the main model's context; lower it (e.g. 128000) if you pin a smaller-context model; `0` omits the variable |
| `--max-thinking-tokens int` | Set `MAX_THINKING_TOKENS` (extended-thinking budget); `0`/omitted = don't set |
| `--max-output-tokens int` | Set `CLAUDE_CODE_MAX_OUTPUT_TOKENS`; `0`/omitted = don't set |

These flags exist only on `coding load` and `coding auth`, and they take
effect when a tool config is written: `coding load <tool>` and
`coding auth reload <tool>`. `coding auth <plan> <key>` only stores the
credential and ignores them. They have no effect on tools other than Claude
Code. Every `load` removes the variables it manages before writing, so tuning
from a previous run is not carried over: pass the flags again each time.

## Key management

```bash
go-z-ai coding auth glm_coding_plan_global YOUR_KEY   # validate and store
go-z-ai coding auth revoke              # clear the stored key, keep the plan choice
go-z-ai coding auth reload <tool>       # re-push stored creds into a tool
go-z-ai coding load <tool> --key OTHER_KEY --plan glm_coding_plan_china  # one-off override
```

`--plan` and `--key` exist on `coding load` and `coding mcp add`; each falls
back to the stored value on its own. With no stored credentials and no
overrides the command fails with `no credentials configured`.

By default, `coding auth` validates a new key against the API before storing
it: a real `GET /models` on the plan's coding endpoint (30 second timeout). A
401 is reported as rejected (`Z.AI rejected the key (401)`); a network error
is reported as a validation failure. Skip validation with `--no-validate` if
you want to store a key offline (e.g. scripting a machine you haven't
network-tested yet).

## MCP servers

The official `@z_ai/coding-helper` has a "manage MCP services" step.
`coding mcp` does the same for Z.AI's four official GLM Coding Plan MCP
servers, registering them in whichever tool you're using:

| Server ID | Kind | What it does |
|---|---|---|
| `zai-mcp-server` | Local, `npx -y @z_ai/mcp-server` | Vision: screenshot OCR, error-screenshot diagnosis, UI-to-artifact and UI diff checks, diagram and chart understanding, image and video analysis ([docs](https://docs.z.ai/devpack/mcp/vision-mcp-server)) |
| `web-search-prime` | Hosted, streamable HTTP | Web search ([docs](https://docs.z.ai/devpack/mcp/search-mcp-server)) |
| `web-reader` | Hosted, streamable HTTP | Fetch and read web pages ([docs](https://docs.z.ai/devpack/mcp/reader-mcp-server)) |
| `zread` | Hosted, streamable HTTP | Read and search GitHub repositories ([docs](https://docs.z.ai/devpack/mcp/zread-mcp-server)) |

```bash
go-z-ai coding mcp add claude-code              # all four, using the stored credentials
go-z-ai coding mcp add opencode --server web-search-prime --server zread
go-z-ai coding mcp add crush --plan glm_coding_plan_china --key OTHER_KEY
go-z-ai coding mcp remove claude-code --server zread
go-z-ai coding mcp status                       # which tools have which servers
```

`--server` takes a server ID, repeatable or comma-separated; the default is
every server the tool supports, and an unknown ID is an error that lists the
valid ones. `add` writes
the entry again if it already exists, which refreshes the key. `remove`
deletes only the named official entries and leaves every other MCP server in
the file alone. `mcp status` reads each tool's MCP file and lists the official
server IDs it finds there.

**Codex takes only the Vision server.** Codex's streamable-HTTP MCP client
treats the empty reply Z.AI's hosted servers send to
`notifications/initialized` as fatal
([openai/codex#14793](https://github.com/openai/codex/issues/14793)), so the
official helper hides them for Codex and `coding mcp add codex` does the
same: it registers Vision by default and refuses a hosted server with that
reason, writing nothing.

The hosted servers are `https://<gateway>/api/mcp/<name>/mcp` with
`<name>` one of `web_search_prime`, `web_reader`, `zread`, authenticated with
`Authorization: Bearer <your key>`. The Vision server runs locally and is
given the key as `Z_AI_API_KEY`, plus `Z_AI_MODE`: `ZAI` for the global plan,
`ZHIPU` for the China plan. That switch decides which platform the Vision
server calls, so a China key under `ZAI` would hit `api.z.ai`.

Calls to these servers draw on your plan's quota. On the credits plan Web
Search, Web Reader and Zread cost 1.2 credits per call
([docs](https://docs.z.ai/devpack/overview)); see
[Accounts & Quota](accounts-and-quota.md).

**Requires Node.js for the Vision server.** It runs via `npx`. Z.AI's own
docs recommend Node.js 22+, while the npm package declares an 18+
requirement. `coding mcp add` warns (doesn't block) if `npx` isn't found on
`PATH` and you selected the Vision server, and `coding doctor` and
`coding mcp status` print the same note, since the config is valid the moment
Node.js becomes available.

**The MCP config file often isn't the same file as your GLM credential.**
Two of the five tools keep MCP servers in a separate file from provider/API
settings, and each tool has its own entry shape:

| Tool | Credential config | MCP config | Entry shape |
|---|---|---|---|
| Claude Code | `~/.claude/settings.json` | `~/.claude.json`, key `mcpServers` | `{type: "stdio", command, args, env}` / `{type: "http", url, headers}` |
| Codex | `~/.codex/config.toml` | same file, table `mcp_servers` | `{type: "local", command, args, env}`; hosted servers not supported |
| OpenCode | `opencode.json` | same file, key `mcp` | `{type: "local", command: [...], environment}` / `{type: "remote", url, headers}` |
| Crush | `crush.json` | same file, key `mcp` | `{type: "stdio", ...}` / `{type: "http", ...}` |
| Factory Droid | `~/.factory/settings.json` | `~/.factory/mcp.json`, key `mcpServers` | as Claude Code, plus `disabled: false` |

## Status and JSON output

```bash
go-z-ai coding status          # stored credentials (key masked) + every tool
go-z-ai coding tools           # IDs, commands, install status, config paths
go-z-ai coding mcp status      # official MCP servers per tool
go-z-ai coding status --format json
```

`status`, `tools` and `mcp status` take `--format json` (default `text`).
`tools` and `mcp status` print an array with one object per tool; `status`
prints `{"credentials": {...}, "tools": [...]}` with the key masked. Each tool
object has `id`, `name`, `command`, `installed`, `config_path`, `configured`,
and, when present, `plan`, `model_map` (Claude Code's tier mapping),
`mcp_servers`, and `error` (a config that couldn't be read, e.g. malformed
JSON; one unreadable tool doesn't stop the others from being reported).

```text
Stored credentials
==================
  Plan: GLM Coding Plan (Global)
  Key:  sk-t****abcd

Coding tools
============
  Claude Code    installed       Z.AI · GLM Coding Plan (Global) · MCP: zai-mcp-server, web-search-prime, web-reader, zread
                 models: haiku=glm-5.3-flash[1m] sonnet=glm-5.3[1m] opus=glm-5.3[1m]
  Codex          installed       Z.AI · GLM Coding Plan (Global) · MCP: zai-mcp-server
  OpenCode       not installed   native config
  Crush          not installed   native config
  Factory Droid  not installed   native config
```

`native config` means the tool's config has no Z.AI plan in it.

## Doctor

```bash
go-z-ai coding doctor
```

Checks that a plan and key are stored, that each supported tool's config can
be read, and that at least one supported tool is installed on `PATH`; it also
notes whether `npx` is available for the Vision MCP server. Each finding is
one line (`✓` ok, `⚠` problem, `ℹ` note). Good first step when something
isn't working.

`doctor` exits with status 1 and `N problem(s) found` when it finds any
problem: no stored credentials, a tool config that cannot be parsed, or no
supported tool on `PATH`. A missing `npx` is only a note and does not change
the exit status, so `doctor` is usable as a CI or provisioning check.

## Compliance & usage policy ⚠️

Z.AI's coding endpoint (`/api/coding/paas/v4`) is restricted by the
[usage policy](https://docs.z.ai/devpack/usage-policy) to "officially
supported tools," and "SDK-based access" is explicitly prohibited. Three
violations result in an account ban. Unidentified third-party clients are
indistinguishable from prohibited access at the server
([pi#4187](https://github.com/earendil-works/pi/issues/4187)). The
[plan overview](https://docs.z.ai/devpack/overview) says the same: the plan is
for officially supported tools.

`go-z-ai` mitigates this in two ways:

1. **The `coding` subcommand wires tools Z.AI documents** (Claude Code,
   Codex, OpenCode, Crush, Factory Droid each have a page under
   `docs.z.ai/devpack/tool/`) into their native config formats — the same
   thing Z.AI's own `@z_ai/coding-helper` does. The wiring itself is the
   supported path; `go-z-ai` just automates it from a Go binary. Which tools
   are on the policy's supported list is Z.AI's call; check the policy page
   for the current list.
2. **Every request `go-z-ai` itself makes carries an identifying
   `User-Agent: go-z-ai/<version>` header** (overridable via
   `Config.UserAgent` for downstream apps, proxies, and MCP servers that need
   their own identifier). This is the minimum hygiene that distinguishes
   go-z-ai from anonymous/prohibited access.

What this does **not** do:

- It does **not** make go-z-ai an "officially supported tool" — only Z.AI can
  confer that status. Until then, using the `coding` subcommand to wire a
  supported tool is the compliant path; using `pkg/client` directly against
  `/api/coding/paas/v4` from a custom integration is at the user's own risk.
- It does **not** spoof or evade detection. The header honestly identifies
  the client; the goal is to be a good citizen, not to hide.

If you are building a downstream tool, proxy, or MCP server on top of
`pkg/client`, set a distinct `Config.UserAgent` (e.g.
`"my-tool/1.0 (go-z-ai)"`) so Z.AI can identify your traffic separately and
so you inherit go-z-ai's good-citizen default rather than weakening it.
