# GitHub repository setup checklist

Files in this repo (CI workflow, Dependabot config, issue/PR templates) are
already in place. The rest is repo **Settings**, which only you can apply —
this is the checklist, current as of 2026.

## 1. Branch protection: import the ruleset

Settings → Rules → Rulesets → New ruleset → **Import a ruleset** →
[`./rulesets/main-branch.json`](rulesets/main-branch.json). It targets `main`
and requires:

- No force-pushes, no branch deletion.
- The CI workflow's two jobs (`Build, vet, and test`, `Vulnerability scan`)
  to pass before merging.

It deliberately does **not** require PR approvals — you're the sole
maintainer today. When you have regular outside contributors, edit the
ruleset (or re-import a modified copy) to add a `pull_request` rule with
`required_approving_review_count: 1`.

Equivalent via the API instead of the UI:

```sh
gh api repos/SamyRai/go-z-ai/rulesets -X POST --input .github/rulesets/main-branch.json
```

## 2. Dependabot alerts + security updates

Settings → Advanced Security (or Security → "Configure" on some plans):

- Enable **Dependabot alerts**.
- Enable **Dependabot security updates**.
- `dependabot.yml` in this repo already handles routine version-update PRs
  (weekly, `gomod` + `github-actions`) — alerts/security-updates are the
  separate vulnerability-driven path and need the toggle above.

## 3. CodeQL (code scanning)

Security → Code security → **Set up code scanning** → **Default setup**.
One click — GitHub picks the right query pack for Go and runs it on every
PR to `main`. No workflow file needed; don't add a custom CodeQL workflow
unless you outgrow default setup's configuration options.

## 4. Secret scanning + push protection

Settings → Advanced Security:

- Enable **Secret scanning**.
- Enable **Push protection** — blocks a push that contains a detected secret
  before it ever reaches the remote. Given this repo's `.env`/API-key
  handling, this is the highest-value single toggle here.

## 5. Repo metadata

- Add a **description** and **topics** (e.g. `zai`, `zhipu`, `glm`, `llm`,
  `go`, `cli`, `sdk`) in the repo's About panel — this is how people find it.
- Set the **default branch** to `main` if it isn't already.
- Consider enabling **squash merge only** (Settings → General → Pull
  Requests) to keep `main`'s history linear given the ruleset above.

## Later, if this grows

- **OpenSSF Scorecard** (`ossf/scorecard-action`) publishing SARIF to the
  Security tab — useful once there are outside contributors judging
  trustworthiness before depending on this as a library.
- **GoReleaser** + GitHub Actions OIDC for signed, SLSA-attested release
  binaries, once you're cutting tagged releases people install directly
  (`go install github.com/SamyRai/go-z-ai@latest` already works without this —
  it's for prebuilt binaries).
