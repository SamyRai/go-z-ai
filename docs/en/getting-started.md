# Getting Started

This gets you from zero to your first `zai-client` command in a couple of minutes.

## 1. Install

**Prerequisites:** Go 1.26.4+, a Z.AI API key ([create one here](https://z.ai/manage-apikey)).

```bash
go install github.com/SamyRai/go-z-ai@latest
```

This puts a `go-z-ai` binary on your `$GOPATH/bin`. **Every example in these
docs uses the shorter name `zai-client`**, so add one of these to your shell
startup before going further:

```bash
# Option A — rename (simplest)
mv "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai-client"

# Option B — alias (keeps the original name)
echo "alias zai-client=go-z-ai" >> ~/.zshrc   # or ~/.bashrc
```

Or build from source with the name you want directly:

```bash
git clone https://github.com/SamyRai/go-z-ai.git
cd go-z-ai
go build -o zai-client .
```

Whichever path you took, confirm `zai-client` resolves and is on your `PATH`:

```bash
zai-client --version
```

The rest of this guide assumes the binary is called `zai-client`.

## 2. Authenticate

Pick whichever fits how you work. They resolve in this priority order (highest wins):

| Method | When to use it |
|---|---|
| `--api-key` flag | One-off calls, scripts, CI |
| `--account <name>` flag | You've registered multiple accounts (see [Accounts & Quota](accounts-and-quota.md)) |
| `ZAI_API_KEY` env var (or `.env` file) | Everyday local shell use — the common case |
| Accounts store's active account | You've run `accounts use <name>` and want it to apply by default |

For a single key, the fastest path:

```bash
export ZAI_API_KEY=your_api_key_here
zai-client validate
```

`validate` makes one real API call and confirms the key works before you go
further.

If your key was issued on Z.AI's China platform (`open.bigmodel.cn`), set
`--region china` (or `ZAI_REGION=china`) so quota / usage, account-info,
agents, and account-type detection route to the right host — without it those
calls hit `api.z.ai` and a China-issued key can fail auth. See
[Accounts & Quota § Regional gateways](accounts-and-quota.md#regional-gateways-apiza--openbigmodelcn)
for the full picture; most chat / embeddings / moderations usage needs
nothing extra (a regular `ZAI_API_KEY` authenticates on both platforms).

## 3. Your first commands

```bash
# See what models you have access to
zai-client models list

# Send a chat completion
zai-client chat create "Explain goroutines in one paragraph"

# Stream the response token-by-token
zai-client chat create "Write a haiku about Go" --stream

# Check your quota (GLM Coding Plan accounts)
zai-client usage quota
```

From here:

- **Full command reference:** [CLI Reference](cli-reference.md)
- **Multiple accounts / quota monitoring:** [Accounts & Quota](accounts-and-quota.md)
- **Wire up Claude Code / OpenCode / Crush / Factory Droid / Cursor to your GLM Coding Plan:** [Coding Tools](coding-tools.md)
- **Using this as a Go library instead of a CLI:** [Library Guide](library-guide.md)
- **Full-screen terminal UI** (chat, models, usage, accounts, coding, media, tools tabs in one place): `zai-client tui`

## Troubleshooting

**"API key is required"** — none of the four methods above resolved a key.
Double check `echo $ZAI_API_KEY`, or pass `--api-key` explicitly to confirm.

**"invalid API key" / HTTP 401** — the key was found but Z.AI rejected it.
Regenerate it at [z.ai/manage-apikey](https://z.ai/manage-apikey).

**"Unknown Model" (error 1211) on `embeddings`/`moderations`/`rerank`/`voice`** —
this is almost always an account-entitlement gate, not a bug: your account's
plan doesn't include that model in its catalog. Run `zai-client models list`
to see what's actually available to your key. See
[Accounts & Quota](accounts-and-quota.md) for the full explanation.

**Something else** — [open an issue](https://github.com/SamyRai/go-z-ai/issues)
with the exact command and error output (redact your key).
