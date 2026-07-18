# Documentation

**English** | [简体中文](../zh/README.md) | [Русский](../ru/README.md)

> Translations live under `docs/<lang>/`. English is the source of truth;
> other locales may lag behind it. Patches that change behavior should update
> `docs/en/` first.

## For users

Using the `zai-client` CLI.

| Doc | Covers |
|---|---|
| [Getting Started](getting-started.md) | Install, authenticate, first commands |
| [CLI Reference](cli-reference.md) | Every command, organized by feature area |
| [Accounts & Quota](accounts-and-quota.md) | Multiple accounts, quota/usage monitoring, regional gateways (api.z.ai / open.bigmodel.cn) |
| [Coding Tools](coding-tools.md) | Wiring Claude Code / OpenCode / Crush / Factory Droid / Cursor to your GLM Coding Plan |

## For developers

Using `pkg/client` as a Go library, or contributing to this repo.

| Doc | Covers |
|---|---|
| [Library Guide](library-guide.md) | Every service, with examples — streaming, function/tool calling, structured output, async polling, the `Region` knob |
| [Error Handling](error-handling.md) | `APIError`, the full error-code table, retry behavior |
| [Architecture](architecture.md) | Package layout, the request facade, regional gateway selection, the live-verification convention |
| [Roadmap & Known Limitations](roadmap.md) | What's unverified, unimplemented, or a known bug — good first-contribution material |
| [Contributing](../../CONTRIBUTING.md) | Before you open a PR — including the live-verification/cassette convention |
| [Security Policy](../../SECURITY.md) | How to report a vulnerability |
| [Changelog](../../CHANGELOG.md) | What shipped and when |

## Repo setup

| File | Covers |
|---|---|
| [.env.example](../../.env.example) | Annotated template for `.env` (`ZAI_API_KEY`, `ZAI_API_BASE_URL`, `ZAI_REGION`, multi-account pointer) |
| [Repo setup checklist](../../.github/SETUP.md) | One-time GitHub settings — branch protection ruleset, Dependabot, CodeQL, secret scanning |

## Quick links

- Main [README](../../README.md) — project overview, install one-liner
- [pkg.go.dev reference](https://pkg.go.dev/github.com/SamyRai/go-z-ai) — generated Go API docs
- [Issues](https://github.com/SamyRai/go-z-ai/issues) · [Discussions](https://github.com/SamyRai/go-z-ai/discussions)
