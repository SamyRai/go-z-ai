# Security Policy

## Supported Versions

This project ships as a rolling `main` branch with tagged releases. Only the
latest tagged release and `main` receive security fixes.

## Reporting a Vulnerability

**Please do not open a public GitHub issue for a security vulnerability.**

Preferred: use [GitHub's private vulnerability reporting](../../security/advisories/new)
(Security tab → "Report a vulnerability") on this repository. This opens a
private advisory visible only to maintainers until a fix is ready.

If you can't use that, email **gigadamer@gmail.com** with:

- A description of the issue and its impact.
- Steps to reproduce (a minimal repro is ideal).
- Any suggested fix, if you have one.

You should get an initial response within a few days. Please don't include
real API keys or account-identifying data in a report — a redacted example is
enough.

## Scope

This is a client for the Z.AI API, not a server. Relevant vulnerability
classes include: credential handling (`.env`, the accounts store, TUI
password prompts), request forgery via a user-controlled `--base-url`,
dependency vulnerabilities (tracked via `govulncheck` and Dependabot in CI),
and unsafe handling of API responses. Vulnerabilities in the upstream Z.AI
API itself are out of scope here — report those to Z.AI directly.
