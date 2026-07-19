// Package sitegen renders the go-z-ai project's static HTML site.
//
// The generator is deliberately dependency-light: it uses only the Go stdlib
// plus github.com/yuin/goldmark for markdown rendering (already transitively
// in go.sum via charm.land/glamour). All templates and assets are embedded
// via embed.FS. Output is plain static HTML/CSS/SVG — no JS framework, no
// web fonts, no external CSS.
//
// Run via `go run ./cmd/sitegen -out site/` or `make site`.
package sitegen

import (
	"bytes"
	"context"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template" // sitemap uses text/template (no HTML escaping needed for an XML file)
	"time"
)

// Options configures a Run.
type Options struct {
	OutDir  string // destination directory (created if missing)
	Owner   string // GitHub owner, e.g. "SamyRai"
	Repo    string // GitHub repo, e.g. "go-z-ai"
	Module  string // Go module path, e.g. "github.com/SamyRai/go-z-ai"
	Name    string // project display name, e.g. "Z.AI API Client"
	Tagline string // one-line description
	// RootDir is the repo root for reading markdown sources. Defaults to ".".
	RootDir string
	// SkipNetwork disables GitHub API calls (offline / CI sandbox).
	SkipNetwork bool
	// SourceDate is the reference instant used for relative-time display
	// ("3h ago") and the sitemap <lastmod>. Defaults to the latest git commit
	// date so builds of the same commit are byte-reproducible. Ignored if
	// zero and git is unavailable (falls back to wall-clock).
	SourceDate time.Time
}

// identRE matches URL-safe GitHub owner/repo identifiers. Used to validate
// values that get interpolated into github.com / api.github.com / pkg.go.dev
// URLs — a typo or malicious value with "/", "?", "#", or whitespace would
// otherwise silently construct wrong links or, worse, a request to an
// unexpected host.
var identRE = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// moduleRE is like identRE but allows "/" (Go module paths are slash-bearing).
var moduleRE = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)

// Run executes the full site generation pipeline.
func Run(ctx context.Context, opts Options) error {
	if opts.OutDir == "" {
		opts.OutDir = "site"
	}
	if opts.RootDir == "" {
		opts.RootDir = "."
	}
	if opts.Owner == "" || opts.Repo == "" {
		return fmt.Errorf("owner and repo are required")
	}
	if !identRE.MatchString(opts.Owner) {
		return fmt.Errorf("owner contains unsupported characters: %q", opts.Owner)
	}
	if !identRE.MatchString(opts.Repo) {
		return fmt.Errorf("repo contains unsupported characters: %q", opts.Repo)
	}
	if opts.Module != "" && !moduleRE.MatchString(opts.Module) {
		return fmt.Errorf("module contains unsupported characters: %q", opts.Module)
	}

	// SourceDate: explicit flag > latest git commit date > wall-clock now.
	// Threading this through rendering makes the build byte-reproducible.
	if opts.SourceDate.IsZero() {
		if commitDate, ok := gitCommitDate(); ok {
			opts.SourceDate = commitDate
		}
	}
	setBuildClock(opts.SourceDate)

	site := &SiteView{
		Name:    firstNonEmpty(opts.Name, "Z.AI API Client"),
		Tagline: firstNonEmpty(opts.Tagline, "A Go CLI, library, and TUI for the Z.AI API"),
		Owner:   opts.Owner,
		Repo:    opts.Repo,
		RepoURL: "https://github.com/" + opts.Owner + "/" + opts.Repo,
		Module:  firstNonEmpty(opts.Module, "github.com/"+opts.Owner+"/"+opts.Repo),
		Commit:  safeGitFirst("rev-parse", "--short", "HEAD"),
	}

	// 1. Load templates + markdown renderer.
	tpl, err := LoadTemplates()
	if err != nil {
		return err
	}

	// 2. Gather dynamic data (degrades gracefully on network failure).
	data := gatherData(ctx, opts)

	// 3. Fresh output dir.
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", opts.OutDir, err)
	}

	// 4. Copy assets (CSS, favicon, robots.txt).
	if err := copyAssets(opts.OutDir); err != nil {
		return err
	}

	// 4a. Generate the chroma syntax-highlighting stylesheet at build time
	// (Catppuccin Mocha for dark, Latte for light). Written next to the
	// other assets and linked from layout.html.
	syntaxCSS, err := GenerateSyntaxCSS()
	if err != nil {
		return fmt.Errorf("generate syntax css: %w", err)
	}
	if err := os.WriteFile(filepath.Join(opts.OutDir, "assets", "syntax.css"), syntaxCSS, 0o644); err != nil {
		return err
	}

	// 5. Write .nojekyll so GitHub Pages serves our raw paths.
	if err := os.WriteFile(filepath.Join(opts.OutDir, ".nojekyll"), nil, 0o644); err != nil {
		return err
	}

	// 6. Render one landing per locale (root README rendered as index.html).
	for _, lang := range localesAll {
		readmePath, _ := readmeFileFor(opts.RootDir, lang)
		body, err := renderMarkdownFile(readmePath)
		if err != nil {
			return fmt.Errorf("render %s README: %w", lang, err)
		}
		page := &Page{
			Title:            "", // landing page has no per-page title; layout shows just the project name
			Description:      site.Tagline,
			Lang:             lang,
			ActiveNav:        "home",
			Body:             body,
			AvailableLocales: LocaleLinksFor("index", lang),
		}
		attachURLs(page, "", lang)
		view := &ViewData{Site: site, Page: page, Data: data}
		html, err := ExecuteTemplate(tpl, "landing.html", view)
		if err != nil {
			return err
		}
		outPath := filepath.Join(opts.OutDir, lang, "index.html")
		if lang == "en" {
			// English gets the bare root /index.html too.
			if err := writeFile(filepath.Join(opts.OutDir, "index.html"), html); err != nil {
				return err
			}
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		if err := writeFile(outPath, html); err != nil {
			return err
		}
	}

	// 7. Render per-doc pages for full-docs locales (en/ru/zh).
	// Iterate the deterministic localesAll slice (not the map) so build
	// output is byte-stable across runs of the same input.
	for _, lang := range fullDocLocales() {
		docDir := filepath.Join(opts.RootDir, "docs", lang)
		entries, err := os.ReadDir(docDir)
		if err != nil {
			continue // locale may not have docs yet
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			if e.Name() == "README.md" {
				continue // already rendered as locale index
			}
			docName := strings.TrimSuffix(e.Name(), ".md")
			src, err := os.ReadFile(filepath.Join(docDir, e.Name()))
			if err != nil {
				log.Printf("sitegen: skipping %s/%s: %v", lang, e.Name(), err)
				continue
			}
			body, err := RenderMarkdown(src)
			if err != nil {
				log.Printf("sitegen: render %s/%s: %v", lang, e.Name(), err)
				continue
			}
			page := &Page{
				Title:            humanizeDocName(docName),
				Description:      "",
				Lang:             lang,
				ActiveNav:        "docs",
				Body:             body,
				AvailableLocales: LocaleLinksFor(docName, lang),
			}
			attachURLs(page, docName, lang)
			view := &ViewData{Site: site, Page: page}
			html, err := ExecuteTemplate(tpl, "doc.html", view)
			if err != nil {
				return err
			}
			outPath := filepath.Join(opts.OutDir, lang, docName+".html")
			if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
				return err
			}
			if err := writeFile(outPath, html); err != nil {
				return err
			}
		}
	}

	// 8. Render meta pages (CHANGELOG, CONTRIBUTING, SECURITY, CODE_OF_CONDUCT) at root.
	for _, meta := range []string{"CHANGELOG", "CONTRIBUTING", "SECURITY", "CODE_OF_CONDUCT"} {
		src, err := os.ReadFile(filepath.Join(opts.RootDir, meta+".md"))
		if err != nil {
			log.Printf("sitegen: meta %s missing, skipping: %v", meta, err)
			continue
		}
		body, err := RenderMarkdown(src)
		if err != nil {
			log.Printf("sitegen: render meta %s: %v", meta, err)
			continue
		}
		page := &Page{
			Title:            humanizeDocName(strings.ToLower(meta)),
			Description:      "",
			Lang:             "en",
			ActiveNav:        "",
			Body:             body,
			AvailableLocales: nil, // meta pages are English-only
		}
		page.Canonical = "/" + meta + ".html"
		view := &ViewData{Site: site, Page: page}
		html, err := ExecuteTemplate(tpl, "doc.html", view)
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(opts.OutDir, meta+".html"), html); err != nil {
			return err
		}
	}

	// 9. Sitemap.
	if err := writeSitemap(opts.OutDir, site); err != nil {
		return err
	}

	return nil
}

// gatherData fetches all dynamic content, tolerating failures.
func gatherData(ctx context.Context, opts Options) *LandingData {
	repoURL := "https://github.com/" + opts.Owner + "/" + opts.Repo
	if opts.SkipNetwork {
		d := &LandingData{Git: CollectGitStats()}
		enrichCommits(d.Git.RecentCommits, repoURL)
		return d
	}
	client := &http.Client{Timeout: 8 * time.Second}
	d := &LandingData{
		Repo: FetchGitHubRepo(ctx, client, opts.Owner, opts.Repo),
		Git:  CollectGitStats(),
	}
	enrichCommits(d.Git.RecentCommits, repoURL)
	rs := FetchReleases(ctx, client, opts.Owner, opts.Repo, 5)
	d.RecentReleases = rs
	if len(rs) > 0 {
		d.LatestVersion = rs[0].TagName
	} else {
		d.LatestVersion = d.Git.LastRelease
	}
	if cl, err := ParseChangelog(filepath.Join(opts.RootDir, "CHANGELOG.md")); err == nil {
		d.ChangelogReleases = cl
	}
	return d
}

func copyAssets(outDir string) error {
	af := AssetFS()
	return fs.WalkDir(af, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		src, err := fs.ReadFile(af, p)
		if err != nil {
			return err
		}
		dst := filepath.Join(outDir, "assets", p)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, src, 0o644)
	})
}

// attachURLs fills Canonical / XDefault / Hreflang for a page given its docname and lang.
func attachURLs(page *Page, docname, lang string) {
	page.Canonical = PageURL(lang, docname)
	page.XDefault = PageURL("en", docname)
	page.HasXDefault = true
	page.Hreflang = HreflangsFor(docname)
}

// renderMarkdownFile reads the file and renders it to HTML. Missing file → empty body.
func renderMarkdownFile(p string) (HTML, error) {
	if p == "" {
		return "", nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return RenderMarkdown(b)
}

// HTML is an alias to html/template.HTML so callers in this package don't
// need to import that module directly.
type HTML = htmltemplate.HTML

// writeFile writes bytes to dst, creating parent dirs.
func writeFile(dst string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

// readmeFileFor resolves the root README path for a given locale.
func readmeFileFor(root, lang string) (string, error) {
	candidates := []string{"README.md"}
	if lang != "en" {
		candidates = []string{
			filepath.Join(root, "README."+lang+".md"),
			filepath.Join(root, "README.md"), // fallback
		}
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", os.ErrNotExist
}

// humanizeDocName turns "getting-started" into "Getting started".
func humanizeDocName(name string) string {
	switch strings.ToLower(name) {
	case "readme", "index", "":
		return "Z.AI API Client"
	case "cli-reference":
		return "CLI reference"
	case "code_of_conduct":
		return "Code of conduct"
	}
	// Hyphen → space, title-case first letter only (matches doc titles like
	// "Accounts & quota", "Coding tools").
	parts := strings.Split(strings.ReplaceAll(name, "_", "-"), "-")
	for i, p := range parts {
		if i == 0 {
			parts[i] = titleCase(p)
		} else {
			parts[i] = strings.ToLower(p)
		}
	}
	return strings.Join(parts, " ")
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// writeSitemap writes a minimal sitemap.xml of all generated pages.
func writeSitemap(outDir string, site *SiteView) error {
	urls := []string{"/"}
	for _, lang := range localesAll {
		urls = append(urls, PageURL(lang, ""))
	}
	for _, lang := range fullDocLocales() {
		for _, doc := range []string{"getting-started", "cli-reference", "accounts-and-quota", "coding-tools", "library-guide", "error-handling", "architecture", "roadmap"} {
			urls = append(urls, PageURL(lang, doc))
		}
	}
	for _, meta := range []string{"CHANGELOG", "CONTRIBUTING", "SECURITY", "CODE_OF_CONDUCT"} {
		urls = append(urls, "/"+meta+".html")
	}

	host := "https://" + site.Owner + ".github.io/" + site.Repo
	const tpl = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
{{- range .URLs }}
  <url><loc>{{ $.Host }}{{ . }}</loc><lastmod>{{ $.Now }}</lastmod></url>
{{- end }}
</urlset>
`
	t := template.Must(template.New("sitemap").Parse(tpl))
	data := struct {
		Host string
		Now  string
		URLs []string
	}{Host: host, Now: buildClock().UTC().Format("2006-01-02"), URLs: urls}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "sitemap.xml"), buf.Bytes(), 0o644)
}

// enrichCommits fills in the GitHub web URL for each commit hash so the
// activity feed can link to the full commit view.
func enrichCommits(commits []Commit, repoURL string) {
	for i := range commits {
		if commits[i].Hash != "" {
			commits[i].URL = repoURL + "/commit/" + commits[i].Hash
		}
	}
}
