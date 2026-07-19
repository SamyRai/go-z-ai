# Site generation

The project ships an auto-generated static HTML site under `site/`, rendered
from the markdown docs + live GitHub data by a tiny Go generator.

## What it does

`cmd/sitegen` (a separate binary in this repo, *not* part of the `go-z-ai`
release) reads:

- The 6 root `README*.md` files → one landing page per locale.
- `docs/{en,ru,zh}/*.md` (27 files) → per-locale doc pages.
- `CHANGELOG.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md` →
  top-level meta pages.
- Live GitHub data (latest release, star count, contributors, recent commits)
  via the unauthenticated GitHub REST API.
- Local `git log` for activity metrics.

…and renders everything through embedded HTML templates into static files at
`site/`:

```text
site/
├── index.html              # English landing (also at /)
├── en/, ru/, zh/           # full-docs locales (landing + 8 doc pages each)
├── de/, tt/, tr/           # README-only locales (landing page only)
├── CHANGELOG.html, CONTRIBUTING.html, …
├── assets/                 # CSS, favicon, robots.txt
└── sitemap.xml
```

## Run it

```bash
make site          # generate into ./site
make site-offline  # same, but skip the live GitHub API calls
make site-serve    # generate + serve on http://localhost:8000
make site-clean    # remove ./site
```

`make site-serve` is the fastest feedback loop for previewing changes — open
<http://localhost:8000/en/> and you'll see the landing page.

## Design choices

**Zero new runtime dependencies.** The generator uses only the Go stdlib plus
[`github.com/yuin/goldmark`](https://github.com/yuin/goldmark), which was
already transitively in `go.sum` via `charm.land/glamour`. Promoting it to a
direct dependency added zero bytes to the release artifacts. There is no Hugo,
no Jekyll, no Node toolchain, no JS framework, no web font.

**Catppuccin Mocha / Latte theme.** The CSS palette is sourced from
[Catppuccin](https://catppuccin.com/palette) (MIT licensed) — already in the
Go dependency tree via `github.com/catppuccin/go`. Dark mode (Mocha) is the
default; light mode (Latte) is auto-selected by `@media (prefers-color-scheme)`
and can be toggled manually via the header button. No CSS framework — about
300 lines of hand-written modern CSS using custom properties, `color-mix()`,
and logical properties.

**Localization.** Pages live at subdirectory permalinks (`/en/`, `/ru/`, …)
matching `docs/<lang>/` one-to-one. Every page emits self-referencing
`hreflang` + alternates + `x-default` → `/en/`. The language switcher is a
plain `<details>` dropdown of `<a>` links — no JavaScript state, deep-linkable,
SEO-correct. For locales that have only a root README (de/tt/tr), the switcher
links those locales back to their landing page; doc-page links for those
locales point to `/en/…` equivalents.

**Graceful degradation.** If the GitHub API is unreachable at build time (rate
limits, sandboxed CI without network), the landing page still renders — it
just shows zeros for stars/releases. Pass `-offline` to skip the network calls
deliberately.

## Deployment

The site is auto-deployed to GitHub Pages on every push to `main` by
[`.github/workflows/pages.yml`](../../.github/workflows/pages.yml). One-time
setup:

1. Repo Settings → Pages → Source: **GitHub Actions** (not "Deploy from a
   branch"). The workflow handles the rest.
2. Confirm the deploy job's environment `github-pages` exists (Actions
   creates it on first run).

Each release also bundles an offline-rendered snapshot of the site via
`release.extra_files` in `.goreleaser.yml`, so `docs-site-<version>.tar.gz`
ships as a release artifact alongside the binaries.

## Adding a new locale

1. Add the new code to `localesAll` in `internal/sitegen/render.go`.
2. Add the code → endonym mapping in `localeLabels`.
3. If you're shipping a full docs tree (not just a README), also add the code
   to `localesWithFullDocs` — the generator will then walk `docs/<lang>/*.md`.
4. Add the language switcher entry to each root `README.<lang>.md` (see the
   existing `**English** | [简体中文]…` line).

## Files

- `cmd/sitegen/main.go` — entrypoint; flags + invokes `sitegen.Run`.
- `internal/sitegen/sitegen.go` — orchestrator.
- `internal/sitegen/render.go` — goldmark setup, template loading, URL helpers.
- `internal/sitegen/data.go` — GitHub API client.
- `internal/sitegen/changelog.go` — CHANGELOG.md parser.
- `internal/sitegen/git.go` — `git log` / `git shortlog` wrappers.
- `internal/sitegen/embed.go` — `//go:embed` for templates + assets.
- `internal/sitegen/templates/*.html` — layout, landing, doc.
- `internal/sitegen/assets/*` — CSS, SVG favicon, robots.txt.
