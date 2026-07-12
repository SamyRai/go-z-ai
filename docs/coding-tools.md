# Coding Tools (GLM Coding Plan)

`zai-client coding` configures third-party coding assistants to use your GLM
Coding Plan instead of their default provider. It's a Go port of Z.AI's
official `@z_ai/coding-helper` ("chelper") CLI, sharing the same credential
file so the two tools can be used interchangeably.

## Supported tools

| Tool | Config file it writes |
|---|---|
| Claude Code | `~/.claude/settings.json` (+ `~/.claude.json` onboarding flag) |
| OpenCode | `~/.config/opencode/opencode.json` |
| Crush | `~/.config/crush/crush.json` |
| Factory Droid | `~/.factory/settings.json` |
| Cursor | OS-specific — `~/Library/Application Support/Cursor/User/settings.json` on macOS, `~/.cursor/settings.json` (or `~/.config/Cursor/User/settings.json`) elsewhere |

Run `zai-client coding tools` to see install status and exact resolved paths
on your machine.

## Plans

| Plan identifier | Endpoint |
|---|---|
| `glm_coding_plan_global` | `https://api.z.ai` |
| `glm_coding_plan_china` | `https://open.bigmodel.cn` |

Pick whichever matches where your GLM Coding Plan subscription lives.

## Quickstart

```bash
# 1. Store and validate your GLM Coding Plan key (one-time)
zai-client coding auth glm_coding_plan_global YOUR_KEY

# 2. Load it into a tool
zai-client coding load claude-code
# tool IDs: claude-code, opencode, crush, factory-droid, cursor
# aliases also work: claude, droid, factory

# 3. Check everything's wired up
zai-client coding status
zai-client coding doctor
```

Credentials live at `~/.chelper/config.yaml` (byte-compatible with the
official Node helper) — `coding auth` writes there once, and `coding load`
reads from it for every tool unless you pass `--key`/`--plan` overrides.

To stop using Z.AI for a tool without losing your stored credential:

```bash
zai-client coding unload claude-code
```

This removes only the Z.AI-specific fields it added; it does not touch the
rest of your existing config file.

## Claude Code: model mapping

The official helper only sets `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_BASE_URL`,
`API_TIMEOUT_MS`, and `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`. This client
goes further by default and also maps Claude Code's model tiers to specific
GLM models via `ANTHROPIC_DEFAULT_*_MODEL`, matching
[Z.AI's documented recommendation](https://docs.z.ai/scenario-example/develop-tools/claude):

| Claude tier | Default GLM model |
|---|---|
| haiku | `glm-4.5-air` |
| sonnet | `glm-4.7` |
| opus | `glm-4.7` |

Override any tier, or opt out entirely:

```bash
zai-client coding auth glm_coding_plan_global YOUR_KEY \
  --sonnet glm-5.2 --opus glm-5.2

zai-client coding auth glm_coding_plan_global YOUR_KEY --no-model-mapping
```

Also configurable, all optional (0/omitted = don't set the env var):

| Flag | Env var it sets | Why you'd use it |
|---|---|---|
| `--auto-compact-window int` | `CLAUDE_CODE_AUTO_COMPACT_WINDOW` | Defaults to 1,000,000 (GLM-5.2's context size); lower it (e.g. 128000) if you're pinned to a 128K-context model |
| `--max-thinking-tokens int` | `MAX_THINKING_TOKENS` | Extended-thinking budget |
| `--max-output-tokens int` | `CLAUDE_CODE_MAX_OUTPUT_TOKENS` | Output cap |

These flags are persistent on the `coding` command, so they apply the same
way to `auth`, `load`, and `reload`.

## Key management

```bash
zai-client coding auth revoke              # clear the stored key, keep the plan choice
zai-client coding auth reload <tool>       # re-push stored creds into a tool
zai-client coding load <tool> --key OTHER_KEY --plan glm_coding_plan_china  # one-off override
```

By default, `coding auth` validates a new key against the API before storing
it (a real `/models` call). Skip that with `--no-validate` if you want to
store a key offline (e.g. scripting a machine you haven't network-tested yet).

## Doctor

```bash
zai-client coding doctor
```

Checks: is a credential stored, does it look well-formed, which supported
tools are installed on `PATH`, and which of those already have a Z.AI
configuration. Good first step when something isn't working.
