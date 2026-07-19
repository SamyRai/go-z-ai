package sitegen

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"regexp"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

// Page is the per-page view model passed to every template.
type Page struct {
	Title              string
	Description        string
	Lang               string
	ActiveNav          string
	Body               template.HTML
	Canonical          string
	XDefault           string
	HasXDefault        bool
	Hreflang           []Hreflang
	AvailableLocales   []LocaleLink
	LangSwitchOpen     bool
	TranslationPending bool
	EnglishEquivHref   string
	OGType             string
	OpenGraph          bool
}

// Hreflang is one <link rel="alternate" hreflang=…>.
type Hreflang struct {
	Lang string
	Href string
}

// LocaleLink is one entry in the language switcher.
type LocaleLink struct {
	Lang  string
	Label string
	Href  string
}

// SiteView is the static site metadata shared by all pages.
type SiteView struct {
	Name    string
	Tagline string
	Owner   string
	Repo    string
	RepoURL string
	Module  string
	Commit  string
}

// ViewData is everything a template sees.
type ViewData struct {
	Site *SiteView
	Page *Page
	Data *LandingData
}

// LandingData is the optional dynamic content shown on the landing page only.
type LandingData struct {
	LatestVersion     string
	Repo              GitHubRepo
	Git               GitStats
	RecentReleases    []GitHubRelease
	ChangelogReleases []ChangelogRelease
}

// localeLabels maps ISO lang code → endonym for the language switcher.
var localeLabels = map[string]string{
	"en": "English",
	"ru": "Русский",
	"zh": "简体中文",
	"de": "Deutsch",
	"tt": "Татарча",
	"tr": "Türkçe",
}

// localesWithFullDocs are the ones with a docs/<lang>/ tree (9 files each).
// Others (de, tt, tr) have only a root README — landing pages only.
var localesWithFullDocs = map[string]bool{"en": true, "ru": true, "zh": true}

// localesAll is every locale with at least a root README.
var localesAll = []string{"en", "ru", "zh", "de", "tt", "tr"}

// fullDocLocales returns the locales that have a docs/<lang>/ tree, in the
// stable order they appear in localesAll. Use this instead of ranging
// localesWithFullDocs directly — Go map iteration is randomized, which would
// make the build output non-deterministic.
func fullDocLocales() []string {
	out := make([]string, 0, len(localesWithFullDocs))
	for _, lang := range localesAll {
		if localesWithFullDocs[lang] {
			out = append(out, lang)
		}
	}
	return out
}

// mdRenderer is the shared goldmark instance.
// chroma highlighter emits CSS classes (no inline styles); the actual colors
// come from assets/syntax.css generated from Catppuccin Mocha (dark) and
// Latte (light) at build time. WithGuessLanguage(true) ensures code blocks
// without an explicit language fence still get the pre.chroma wrapper so
// styling stays consistent across tagged and untagged blocks.
//
// WithHardWraps is intentionally OFF — docs authors expect soft-wrapped
// paragraphs to flow, not single newlines to become <br>.
//
// Raw inline HTML in source markdown is escaped (no WithUnsafe): the output
// of mdRenderer is then passed through the docSanitizer (below) so any inline
// HTML that does survive is confined to a strict allowlist. This makes docs
// contributions a non-XSS surface — important because the repo accepts
// outside PRs and the rendered HTML ships to the public site.
var mdRenderer = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Linkify,
		extension.TaskList,
		extension.Typographer,
		highlighting.NewHighlighting(
			highlighting.WithStyle("catppuccin-mocha"),
			highlighting.WithGuessLanguage(true),
			highlighting.WithFormatOptions(
				chromahtml.WithClasses(true),
				chromahtml.TabWidth(4),
				chromahtml.WithLineNumbers(false),
			),
		),
	),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
)

// linkFixRE matches href values that need rewriting for the static site.
// Two regexes, applied in sequence:
//
//	linkMdExtRE  — hrefs ending in `.md` (with optional #anchor or ?query):
//	               `foo.md`, `foo.md#x`, `../foo.md`, `docs/en/foo.md`, `README.md`
//	linkBareName — bare hrefs with no extension and a single path component
//	               matching a known doc name: `getting-started`, `cli-reference`, …
//
// External URLs, mailto:, anchor-only (#…), and already-.html links are
// untouched by both regexes.
var (
	// linkMdExtRE: capture (1) `href="`, (2) optional path-prefix ending in `/`
	// (so the basename is cleanly captured separately), (3) basename, (4) trailer
	// (quote or #anchor or ?query then quote). The prefix capture is anchored to
	// end in `/` to prevent greedy backtracking from eating the basename.
	linkMdExtRE  = regexp.MustCompile(`(href=")((?:[^"#?]*/)*)([A-Za-z0-9._-]+)\.md(["#?])`)
	linkBareName = regexp.MustCompile(`(href=")((?:\.\./)*)([A-Za-z0-9._-]+)(["#?])`)

	// leadingCommentRE matches a leading "/* … */" comment in a chroma CSS
	// line, including trailing whitespace, so we can strip it before prefixing
	// the selector with a theme scope.
	leadingCommentRE = regexp.MustCompile(`^\s*/\*[^*]*\*/\s*`)
)

// knownDocBases is the set of names that exist as <name>.html in the site.
// Used to rewrite bare-name links (no extension) emitted by the markdown
// source convention `[X](getting-started)` → works on GitHub, broken in
// static HTML.
var knownDocBases = map[string]bool{
	"getting-started":    true,
	"cli-reference":      true,
	"accounts-and-quota": true,
	"coding-tools":       true,
	"library-guide":      true,
	"error-handling":     true,
	"architecture":       true,
	"roadmap":            true,
	"site-generation":    true,
	"CHANGELOG":          true,
	"CONTRIBUTING":       true,
	"SECURITY":           true,
	"CODE_OF_CONDUCT":    true,
}

// rewriteLinks takes goldmark-rendered HTML and rewrites internal markdown
// links to their static-HTML equivalents so links resolve in the generated
// site. GitHub renders `accounts-and-quota.md` as a working link; our static
// site needs `accounts-and-quota.html`.
//
// Three classes of rewrite:
//  1. `foo.md` (single name in same dir)  → `foo.html`
//  2. `getting-started` (bare known name) → `getting-started.html`
//  3. Any path with `../` segments or `docs/<lang>/...` prefix — these are
//     repo-tree-relative paths that don't match the site's URL layout. We
//     resolve them to the site-root-absolute URL of the target's basename.
//     Example: from inside docs/ru/architecture.md, `../../CONTRIBUTING.md`
//     → `/CONTRIBUTING.html`.
func rewriteLinks(html []byte) []byte {
	// Pass 1: `.md` (with optional anchor) → `.html`. README.md collapses to
	// the directory (handled below in the path-resolution pass for non-trivial
	// cases; for simple same-dir README.md refs we leave the link pointing at
	// the current directory by stripping the README.md component).
	html = linkMdExtRE.ReplaceAllFunc(html, func(m []byte) []byte {
		g := linkMdExtRE.FindSubmatch(m)
		href, prefix, name, trailer := g[1], g[2], g[3], g[4]
		// Multi-segment path (prefix has slashes) → resolve via basename.
		if len(prefix) > 0 {
			if string(name) == "README" {
				// README referenced cross-directory — leave the link as the
				// markdown source had it (broken on the static site, but the
				// rewriter can't infer intent here).
				return m
			}
			return rewriteToSiteRoot(href, name, trailer)
		}
		if string(name) == "README" {
			return []byte(string(href) + string(trailer))
		}
		return []byte(string(href) + string(name) + ".html" + string(trailer))
	})
	// Pass 2: bare names like `getting-started` (no extension) → `.html`.
	html = linkBareName.ReplaceAllFunc(html, func(m []byte) []byte {
		g := linkBareName.FindSubmatch(m)
		href, updirs, name, trailer := g[1], g[2], g[3], g[4]
		if len(updirs) > 0 {
			// Cross-directory reference — defer to site-root resolution.
			if knownDocBases[string(name)] {
				return rewriteToSiteRoot(href, name, trailer)
			}
			return m
		}
		if !knownDocBases[string(name)] {
			return m
		}
		return []byte(string(href) + string(name) + ".html" + string(trailer))
	})
	return html
}

// rewriteToSiteRoot returns an href=…  snippet pointing the link at the
// site-root-absolute URL for a given basename. Meta docs (CHANGELOG etc.)
// live at /<NAME>.html; everything else is treated as a docs page at /en/.
func rewriteToSiteRoot(hrefOpen, name, trailer []byte) []byte {
	n := string(name)
	var dst string
	switch n {
	case "CHANGELOG", "CONTRIBUTING", "SECURITY", "CODE_OF_CONDUCT":
		dst = "/" + n + ".html"
	case "README":
		dst = "/en/"
	default:
		dst = "/en/" + n + ".html"
	}
	return append(append(append([]byte{}, hrefOpen...), []byte(dst)...), trailer...)
}

// docSanitizer is the allowlist HTML policy applied to goldmark output.
// Built on top of bluemonday's StrictPolicy (which allows no markup by
// default) and re-enables the subset needed for rendered docs:
//   - standard prose tags (p, headings, lists, blockquote, emphasis, …)
//   - GFM: tables, task list inputs (disabled), strikethrough
//   - code: pre/code/span (chroma emits <span class="…">)
//   - inline semantic tags occasionally useful in docs: kbd, sub, sup, abbr,
//     mark, details, summary
//   - links and images with safe attributes only
//
// No scripts, iframes, styles, event handlers, or class values outside the
// chroma-highlighting / syntax-css contract.
var docSanitizer = buildDocSanitizer()

func buildDocSanitizer() *bluemonday.Policy {
	p := bluemonday.StrictPolicy()

	// Prose + structure.
	p.AllowElements(
		"p", "br", "hr", "blockquote",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"ul", "ol", "li", "dl", "dt", "dd",
		"strong", "em", "b", "i", "s", "del", "ins", "mark",
		"span", "div",
		"sub", "sup", "abbr", "kbd", "small",
		"details", "summary",
	)
	// Code blocks (goldmark) + chroma highlighting (class-based).
	p.AllowElements("pre", "code", "tt")
	// GFM tables.
	p.AllowTables()

	// Links and images — safe URL schemes and a short attribute allowlist.
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("src", "alt", "title").OnElements("img")
	p.AllowAttrs("name").OnElements("a")
	p.AllowStandardURLs()
	// Rel/target hardening for external links.
	p.RequireNoFollowOnLinks(true)
	p.RequireNoReferrerOnLinks(true)
	p.AddTargetBlankToFullyQualifiedLinks(true)

	// Headings and table cells get align + id (goldmark auto-heading IDs).
	p.AllowAttrs("id").OnElements("h1", "h2", "h3", "h4", "h5", "h6")
	p.AllowAttrs("align").OnElements("th", "td", "col", "colgroup", "table")

	// chroma emits <span class="k">, <span class="chroma">, etc. Allow the
	// class attribute only on inline/code spans so prose can't pick up
	// arbitrary styling hooks.
	p.AllowAttrs("class").OnElements("span", "code", "pre", "div")

	// GFM task lists render disabled checkboxes inside <li>.
	p.AllowAttrs("disabled", "type").OnElements("input")

	return p
}

// sanitizeHTML runs a doc through the allowlist policy. Defense-in-depth on
// top of goldmark's escaping of raw inline HTML: even if a future extension
// or renderer regression lets markup through, the policy strips it.
func sanitizeHTML(in []byte) []byte {
	return docSanitizer.SanitizeBytes(in)
}

// RenderMarkdown converts markdown bytes to HTML. Used by both landing and doc
// rendering paths. Output is sanitized via docSanitizer and safe to embed as
// template.HTML.
func RenderMarkdown(src []byte) (template.HTML, error) {
	var buf bytes.Buffer
	if err := mdRenderer.Convert(src, &buf); err != nil {
		return "", err
	}
	out := rewriteLinks(buf.Bytes())
	out = sanitizeHTML(out)
	return template.HTML(out), nil
}

// GenerateSyntaxCSS produces the theme-aware CSS for chroma's class-based
// highlighting. The Mocha (dark) ruleset applies by default; the Latte (light)
// ruleset applies under [data-theme="light"] or when the OS prefers light.
// All chroma rules are scoped under `pre.chroma` so the bare `.bg` and
// `.chroma` top-level selectors can't collide with anything else on the page.
func GenerateSyntaxCSS() ([]byte, error) {
	formatter := chromahtml.New(chromahtml.WithClasses(true))
	var out bytes.Buffer

	dark, err := renderScopedTheme(formatter, "catppuccin-mocha", "")
	if err != nil {
		return nil, err
	}
	light, err := renderScopedTheme(formatter, "catppuccin-latte", `[data-theme="light"] `)
	if err != nil {
		return nil, err
	}

	fmt.Fprintln(&out, "/* Auto-generated by cmd/sitegen — Catppuccin Mocha (dark default), scoped under pre.chroma. */")
	out.Write(dark)
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "/* Auto-generated by cmd/sitegen — Catppuccin Latte (explicit light override). */")
	out.Write(light)
	return out.Bytes(), nil
}

// renderScopedTheme runs chroma's WriteCSS for the named style and scopes
// every rule under `pre.chroma` (and optionally a theme prefix like
// `[data-theme="light"] `). This prevents chroma's bare `.bg` and `.chroma`
// top-level selectors from colliding with other CSS on the page.
//
// Chroma emits rules like:
//
//	/* Background */ .bg { color: ...; background-color: ...; }
//	/* PreWrapper */ .chroma { color: ...; background-color: ...; }
//	/* Keyword */ .chroma .k { color: ... }
//
// We rewrite the selector list to:
//
//	pre.chroma { ... }            (was .bg and .chroma — merged)
//	pre.chroma .k { ... }         (was .chroma .k — descendant stays)
func renderScopedTheme(formatter *chromahtml.Formatter, styleName, themePrefix string) ([]byte, error) {
	style := chromastyles.Get(styleName)
	if style == nil {
		return nil, fmt.Errorf("chroma style %s not found", styleName)
	}
	var buf bytes.Buffer
	if err := formatter.WriteCSS(&buf, style); err != nil {
		return nil, fmt.Errorf("writecss %s: %w", styleName, err)
	}

	var scoped bytes.Buffer
	for _, line := range bytes.Split(buf.Bytes(), []byte("\n")) {
		braceIdx := bytes.IndexByte(line, '{')
		if braceIdx < 0 {
			continue // skip blank lines / standalone comments
		}
		head := leadingCommentRE.ReplaceAll(line[:braceIdx], []byte(""))
		body := line[braceIdx:]

		// Split into individual selectors and rewrite each one.
		sels := bytes.Split(head, []byte(","))
		var rewritten []string
		for _, sel := range sels {
			s := strings.TrimSpace(string(sel))
			if s == "" {
				continue
			}
			// Replace bare `.bg` or `.chroma` (top-level element selectors)
			// with `pre.chroma` — they target the same <pre> element.
			if s == ".bg" || s == ".chroma" {
				rewritten = append(rewritten, themePrefix+"pre.chroma")
				continue
			}
			// `.chroma .xxx` → `<themePrefix>pre.chroma .xxx`
			if strings.HasPrefix(s, ".chroma ") {
				rewritten = append(rewritten, themePrefix+"pre.chroma "+s[len(".chroma "):])
				continue
			}
			// Any other selector — prefix with theme + pre.chroma scope
			// to be safe.
			rewritten = append(rewritten, themePrefix+"pre.chroma "+s)
		}
		if len(rewritten) == 0 {
			continue
		}
		scoped.WriteString(strings.Join(rewritten, ", "))
		scoped.Write(body)
		scoped.WriteByte('\n')
	}
	return scoped.Bytes(), nil
}

// LoadTemplates parses the embedded templates into a base *template.Template.
// The base holds layout.html (which defines the page chrome and expects a
// "content" block) plus the shared funcs; per-page templates (landing.html,
// doc.html) are cloned from it and provide their own "content" block via
// ExecutePage.
func LoadTemplates() (*template.Template, error) {
	tfs := TemplateFS()
	root := template.New("").Funcs(template.FuncMap{
		"safeHTML": func(v any) template.HTML {
			switch x := v.(type) {
			case template.HTML:
				return x
			case string:
				return template.HTML(x)
			default:
				return ""
			}
		},
		"default": func(d, v any) any {
			if isEmpty(v) {
				return d
			}
			return v
		},
		"dict":         dict,
		"langBase":     langBase,
		"highlight":    highlightCode,
		"relativeTime": relativeTime,
		"lower":        strings.ToLower,
		// T is a placeholder so templates parse cleanly; ExecuteTemplate
		// replaces it with the locale-specific translateFunc on each clone.
		"T": func(key string, args ...any) string { return key },
	})
	// Walk every .html under the templates FS and parse into root.
	err := fs.WalkDir(tfs, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".html") {
			return nil
		}
		b, err := fs.ReadFile(tfs, p)
		if err != nil {
			return err
		}
		_, err = root.New(p).Parse(string(b))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	return root, nil
}

// ExecuteTemplate renders the named page template (landing.html or doc.html)
// against view data. It clones the base template so the page's "content"
// block doesn't leak into other page renders.
func ExecuteTemplate(t *template.Template, name string, vd *ViewData) ([]byte, error) {
	clone, err := t.Clone()
	if err != nil {
		return nil, fmt.Errorf("clone templates: %w", err)
	}
	// Register the per-locale T function on the clone so templates can use
	// {{ T "key" }} to pull translated strings. Falls back to English for
	// missing keys, warns to stderr at build time.
	bundle := LoadLocale(vd.Page.Lang)
	clone.Funcs(template.FuncMap{"T": translateFunc(bundle)})

	// Re-parse the requested page template into the clone so its "content"
	// block is the one that layout.html invokes via {{ block "content" . }}.
	pageSrc, err := fs.ReadFile(TemplateFS(), name)
	if err != nil {
		return nil, fmt.Errorf("missing page template %s: %w", name, err)
	}
	if _, err := clone.New(name).Parse(string(pageSrc)); err != nil {
		return nil, fmt.Errorf("re-parse %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := clone.ExecuteTemplate(&buf, name, vd); err != nil {
		return nil, fmt.Errorf("execute %s: %w", name, err)
	}
	return buf.Bytes(), nil
}

func isEmpty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case template.HTML:
		return x == ""
	default:
		return false
	}
}

// dict builds a map[string]any from k/v pairs, for use inside templates
// where you need to construct a small dict (e.g. the language-switcher label
// lookup). Odd arg count → final value is "".
func dict(args ...any) map[string]any {
	m := make(map[string]any, len(args)/2+1)
	for i := 0; i+1 < len(args); i += 2 {
		key, _ := args[i].(string)
		m[key] = args[i+1]
	}
	if len(args)%2 == 1 {
		// dangling key with no value
		key, _ := args[len(args)-1].(string)
		m[key] = ""
	}
	return m
}

// langBase returns the site-root-relative base URL for a locale's docs:
// "/en/" for English, "/<lang>/" for other locales with a full docs tree.
// README-only locales (de, tt, tr) have no doc pages of their own, so their
// docs links point at the English versions — this function returns "/en/"
// for them so templates can build locale-aware links uniformly.
func langBase(lang string) string {
	if lang == "" {
		return "/en/"
	}
	if localesWithFullDocs[lang] {
		return "/" + lang + "/"
	}
	return "/en/"
}

// highlightCode is a template function that syntax-highlights a raw code
// string for use in template-authored (non-markdown) <pre> blocks. It uses
// chroma's class-based formatter (matching the markdown pipeline) and
// PreventSurroundingPre(true) because the template owns the <pre> wrapper.
// Usage in templates:  <pre class="chroma"><code>{{ highlight .Src "go" }}</code></pre>
func highlightCode(src, lang string) template.HTML {
	l := lexers.Get(lang)
	if l == nil {
		l = lexers.Analyse(src)
	}
	if l == nil {
		l = lexers.Fallback
	}
	l = chroma.Coalesce(l)
	it, err := l.Tokenise(nil, src)
	if err != nil {
		return template.HTML(template.HTMLEscapeString(src))
	}
	f := chromahtml.New(
		chromahtml.WithClasses(true),
		chromahtml.PreventSurroundingPre(true),
		chromahtml.TabWidth(4),
	)
	var buf bytes.Buffer
	if err := f.Format(&buf, chromastyles.Get("catppuccin-mocha"), it); err != nil {
		return template.HTML(template.HTMLEscapeString(src))
	}
	return template.HTML(buf.String())
}

// buildClock is the reference instant used for relative-time rendering
// ("3h ago") and for the sitemap <lastmod>. It is set once per Run to make
// output byte-reproducible across builds of the same commit (using
// time.Since(time.Now()) here would make two builds of identical input
// produce different HTML). Defaults to wall-clock time so callers that don't
// set it behave as before.
var buildClock = time.Now

// setBuildClock configures the reference instant for relative-time rendering.
// Called once at the start of Run with the chosen SourceDate.
func setBuildClock(t time.Time) {
	if t.IsZero() {
		buildClock = time.Now
		return
	}
	buildClock = func() time.Time { return t }
}

// relativeTime renders a time.Time as a human-friendly relative string:
// "just now", "3h ago", "2d ago", "2026-07-19". Used in the activity feed.
func relativeTime(t any) string {
	var tm time.Time
	switch v := t.(type) {
	case time.Time:
		tm = v
	default:
		return ""
	}
	if tm.IsZero() {
		return ""
	}
	d := buildClock().Sub(tm)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return tm.Format("2006-01-02")
	}
}

// PageURL returns the absolute site path for a (locale, docname) pair.
// English also lives at /en/ (we emit /index.html as a duplicate for
// convenience, but canonical URLs always carry the locale prefix so links
// work uniformly across locales). Doc pages get a ".html" suffix so the link
// resolves without server-side content negotiation.
func PageURL(lang, docname string) string {
	if lang == "" {
		lang = "en"
	}
	prefix := "/" + lang + "/"
	if docname == "" || docname == "index" {
		return prefix
	}
	return prefix + docname + ".html"
}

// LocaleLinksFor returns the language switcher entries for a given doc and
// current locale. docname is "" for the landing. If a locale has no equivalent
// (e.g. a full-docs page viewed from tt, which only has a README), the link
// points to that locale's landing page instead.
func LocaleLinksFor(docname, currentLang string) []LocaleLink {
	out := make([]LocaleLink, 0, len(localesAll))
	for _, lang := range localesAll {
		// Doc availability: if it's a doc page and the locale lacks docs, link to landing.
		if docname != "" && docname != "index" && !localesWithFullDocs[lang] {
			out = append(out, LocaleLink{Lang: lang, Label: localeLabels[lang], Href: PageURL(lang, "")})
			continue
		}
		out = append(out, LocaleLink{Lang: lang, Label: localeLabels[lang], Href: PageURL(lang, docname)})
	}
	return out
}

// HreflangsFor returns hreflang entries mirroring LocaleLinksFor but in the
// hreflang {lang, href} shape.
func HreflangsFor(docname string) []Hreflang {
	out := make([]Hreflang, 0, len(localesAll))
	for _, lang := range localesAll {
		if docname != "" && docname != "index" && !localesWithFullDocs[lang] {
			out = append(out, Hreflang{Lang: lang, Href: PageURL(lang, "")})
			continue
		}
		out = append(out, Hreflang{Lang: lang, Href: PageURL(lang, docname)})
	}
	return out
}
