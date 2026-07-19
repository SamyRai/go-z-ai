# GitHub repository setup checklist

Files in this repo (CI workflow, Dependabot config, issue/PR templates) are
already in place. The rest is repo **Settings**, which only you can apply —
this is the checklist, current as of 2026. Items marked ✅ are already
applied; ⬜ are still on you.

## 1. Branch protection: import the ruleset ✅

Settings → Rules → Rulesets → New ruleset → **Import a ruleset** →
[`./rulesets/main-branch.json`](rulesets/main-branch.json). It targets `main`
and requires:

- No force-pushes, no branch deletion.
- The CI workflow's jobs (`Build, vet, and test`, `Vulnerability scan`) to
  pass before merging.

It deliberately does **not** require PR approvals — you're the sole
maintainer today. When you have regular outside contributors, edit the
ruleset (or re-import a modified copy) to add a `pull_request` rule with
`required_approving_review_count: 1`.

Equivalent via the API instead of the UI:

```sh
gh api repos/SamyRai/go-z-ai/rulesets -X POST --input .github/rulesets/main-branch.json
```

## 2. Dependabot alerts + security updates ⬜

Settings → Advanced Security (or Security → "Configure" on some plans):

- Enable **Dependabot alerts**.
- Enable **Dependabot security updates**.
- `dependabot.yml` in this repo already handles routine version-update PRs
  (weekly, `gomod` + `github-actions`) — alerts/security-updates are the
  separate vulnerability-driven path and need the toggle above.
- `.github/workflows/dependency-review.yml` blocks PRs that introduce a
  dependency with a moderate-or-worse vulnerability. Requires the
  dependency graph (auto-on for public repos; otherwise Settings →
  Code security → Dependency graph).

## 3. CodeQL (code scanning) ✅ (workflow) / ⬜ (settings gate)

Two parts:

- **Workflow** — `.github/workflows/codeql.yml` runs the CodeQL `go` analysis
  on push/PR to `main` plus a weekly sweep, with the `security-extended` +
  `security-and-quality` query suites enabled (broader than default setup).
  All actions in it are pinned by SHA.
- **Settings gate ⬜** — Security → Code security → **Set up code scanning** →
  **Default setup**, **or** "Advanced setup" → point at the `codeql.yml`
  workflow. Either way, alerts only start surfacing in the Security tab once
  the toggle is on. Until then the workflow still runs, but results stay in
  the run logs.

> ℹ️ CodeQL Action v3 is being deprecated in December 2026 (GHES 3.19 EOL).
> All CodeQL steps here already use `@v4`.

## 4. Secret scanning + push protection ⬜ (public-launch gate)

Settings → Advanced Security:

- Enable **Secret scanning**.
- Enable **Push protection** — blocks a push that contains a detected secret
  before it ever reaches the remote. Given this repo's `.env`/API-key
  handling, this is the highest-value single toggle here.

> **Public-launch gate.** Turn this on **before** flipping the repo from
> private to public. It's the single best defense against accidentally
> committing a Z.AI key. The repo-level `.gitignore` already excludes `.env`,
> but push protection catches keys pasted into source/docs/cassettes too.

## 5. Repo metadata ✅

- **Description** and **topics** set in the repo's About panel (topics
  include `zai`, `zhipu`, `glm`, `llm`, `go`, `cli`, `sdk`, plus the
  original `ai`, `anthropic`, `api`, `client`, `integration`, `openai`).
- **Default branch** is `main`.
- Consider enabling **squash merge only** (Settings → General → Pull
  Requests) to keep `main`'s history linear given the ruleset above.
- **Homepage** points at the docs index.
- **Discussions** are enabled; the `Releases` category is used by GoReleaser
  to announce new versions.

## 6. Releases ✅

Tag-driven via `.github/workflows/release.yml` + `.goreleaser.yml`. Pushing a
`v*` tag triggers GoReleaser, which builds for
`linux/amd64, linux/arm64, darwin/amd64, darwin/arm64`, attaches an SBOM,
signs artifacts keyless with cosign/sigstore (OIDC), and publishes a GitHub
Release with an auto-generated changelog. `pkg.go.dev` picks up the tag
within minutes.

First release: `v0.1.0` (2026-07-19).

## 7. OpenSSF Scorecard ✅

`.github/workflows/scorecard.yml` runs `ossf/scorecard-action` weekly
(Monday ~05:17 UTC), publishes results to the Security tab (SARIF), and
exposes them at
`api.securityscorecards.dev/github.com/SamyRai/go-z-ai/badge` — which is the
badge surfaced in the README.

All third-party actions across every workflow are pinned by 40-char commit
SHA with a `# vX.Y.Z` version comment. This is what Scorecard's
`Pinned-Dependencies` check rewards (tag pins like `@v5` are mutable and
score 0); Dependabot keeps them current via its `github-actions` ecosystem
(see `dependabot.yml`).

> ℹ️ Replaces the now-sunset Go Report Card badge. goreportcard.com shut down
> its badge service on July 1, 2026 after 11 years; the badge endpoint now
> returns a static "retired" placeholder for every repo regardless of grade.

## Later, if this grows

- **CODEOWNERS** — add when regular outside contributors arrive (today the
  ruleset deliberately skips required reviews).
