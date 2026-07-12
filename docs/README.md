# Documentation

## For users

Using the `zai-client` CLI.

| Doc | Covers |
|---|---|
| [Getting Started](getting-started.md) | Install, authenticate, first commands |
| [CLI Reference](cli-reference.md) | Every command, organized by feature area |
| [Accounts & Quota](accounts-and-quota.md) | Multiple accounts, quota/usage monitoring, the China-platform key note |
| [Coding Tools](coding-tools.md) | Wiring Claude Code / OpenCode / Crush / Factory Droid / Cursor to your GLM Coding Plan |

## For developers

Using `pkg/client` as a Go library, or contributing to this repo.

| Doc | Covers |
|---|---|
| [Library Guide](library-guide.md) | Every service, with examples — streaming, function calling, structured output, async polling |
| [Error Handling](error-handling.md) | `APIError`, the full error-code table, retry behavior |
| [Architecture](architecture.md) | Package layout, the request facade, why some services hit a different host, the live-verification convention |
| [Roadmap & Known Limitations](roadmap.md) | What's unverified, unimplemented, or a known bug — good first-contribution material |
| [Contributing](../CONTRIBUTING.md) | Before you open a PR — including the live-verification/cassette convention |
| [Security Policy](../SECURITY.md) | How to report a vulnerability |
| [Changelog](../CHANGELOG.md) | What shipped and when |

## Quick links

- Main [README](../README.md) — project overview, install one-liner
- [pkg.go.dev reference](https://pkg.go.dev/github.com/SamyRai/go-z-ai) — generated Go API docs
- [Issues](https://github.com/SamyRai/go-z-ai/issues) · [Discussions](https://github.com/SamyRai/go-z-ai/discussions)
