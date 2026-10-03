# Getting Started

This gets you from zero to your first `go-z-ai` command in a couple of minutes.

## 1. Install

**Prerequisites:** Go 1.26.4+, a Z.AI API key ([create one here](https://z.ai/manage-apikey/apikey-list)).

```bash
go install github.com/SamyRai/go-z-ai@latest
```

The binary installs as `go-z-ai`. Optional short alias:

```bash
ln -s "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai"
```

Or build from source directly:

```bash
git clone https://github.com/SamyRai/go-z-ai.git
cd go-z-ai
go build -o go-z-ai .
```

Whichever path you took, confirm `go-z-ai` resolves and is on your `PATH`:

```bash
go-z-ai --version
```

The rest of this guide assumes the binary is called `go-z-ai`.

## 2. Authenticate

Pick whichever fits how you work. They resolve in this priority order (highest wins):

| Method | When to use it |
|---|---|
| `--api-key` flag | One-off calls, scripts, CI |
| `--account <name>` flag | You've registered multiple accounts (see [Accounts & Quota](accounts-and-quota.md)) |
| `ZAI_API_KEY` env var (or a `.env` file in the current directory) | Everyday local shell use — the common case |
| Accounts store's active account | You've run `accounts use <name>` and want it to apply by default |

For a single key, the fastest path:

```bash
export ZAI_API_KEY=your_api_key_here
go-z-ai validate
```

`validate` makes one free request (it lists models) and confirms the key works
before you go further.

These environment variables are read (each has a matching flag):

| Variable | Flag | Purpose |
|---|---|---|
| `ZAI_API_KEY` | `--api-key` | Your Z.AI API key |
| `ZAI_REGION` | `--region` | Regional gateway: `global` (api.z.ai, default) or `china` (open.bigmodel.cn) |
| `ZAI_API_BASE_URL` | `--base-url` | Override the chat/PaaS API root only (default: the region's) |
| `ZAI_CHINA_API_KEY` | `--china-api-key` | Separate open.bigmodel.cn key for Embeddings/Moderations (falls back to `ZAI_API_KEY`) |

`ZAI_MONITOR_TIMEZONE` (`--monitor-timezone`) is also read; it only matters for
quota/usage output.

If your key was issued on Z.AI's China platform (`open.bigmodel.cn`), set
`--region china` (or `ZAI_REGION=china`). The region selects the host for every
endpoint — chat, quota / usage, account, agents, and account-type detection —
so without it those calls go to `api.z.ai`, where a China-issued key can fail
auth. `accounts add` detects the region for you when you register a key (with
`--type` it skips detection, so add `--region china` yourself). Embeddings and
Moderations always use `open.bigmodel.cn`. See
[Accounts & Quota § Regional gateways](accounts-and-quota.md#regional-gateways-apiza--openbigmodelcn)
for the full picture.

## 3. Your first commands

```bash
# See what models you have access to (context, prices, capabilities from the catalog)
go-z-ai models list

# Send a chat completion (uses the catalog's default chat model)
go-z-ai chat create "Explain goroutines in one paragraph"

# Stream the response token-by-token
go-z-ai chat create "Write a haiku about Go" --stream

# Detect whether the key is Coding Plan or pay-as-you-go, and on which gateway (free)
go-z-ai account detect

# Check that the key can spend (free for Coding Plan keys; one minimal billed
# request for pay-as-you-go keys), then your quota (Coding Plan accounts only)
go-z-ai account status
go-z-ai usage quota
```

Add `--format json` to most commands for machine-readable output; progress
messages go to stderr, so stdout stays clean.

From here:

- **Full command reference:** [CLI Reference](cli-reference.md)
- **Multiple accounts / quota monitoring:** [Accounts & Quota](accounts-and-quota.md)
- **Wire up Claude Code / OpenCode / Crush / Factory Droid to your GLM Coding Plan:** [Coding Tools](coding-tools.md)
- **Using this as a Go library instead of a CLI:** [Library Guide](library-guide.md)
- **Full-screen terminal UI** (chat, models, usage, accounts, coding, media, tools tabs in one place): `go-z-ai tui`

## Troubleshooting

**"API key is required"** — none of the four methods above resolved a key.
Double check `echo $ZAI_API_KEY`, or pass `--api-key` explicitly to confirm.

**"invalid API key" / HTTP 401** — the key was found but Z.AI rejected it.
Regenerate it at [z.ai/manage-apikey](https://z.ai/manage-apikey/apikey-list).

**"Unknown Model" (error 1211) on `embeddings`/`moderations`/`rerank`/`voice`** —
this is almost always an account-entitlement gate, not a bug: your account's
plan doesn't include that model in its catalog. Run `go-z-ai models list`
to see what's actually available to your key. See
[Accounts & Quota](accounts-and-quota.md) for the full explanation.

**Something else** — [open an issue](https://github.com/SamyRai/go-z-ai/issues)
with the exact command and error output (redact your key).
