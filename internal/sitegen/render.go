package sitegen

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
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
	Name     string
	Tagline  string
	Owner    string
	Repo     string
	RepoURL  string
	Module   string
	Commit   string
}

// ViewData is everything a template sees.
type ViewData struct {
	Site *SiteView
	Page *Page
	Data *LandingData
}

// LandingData is the optional dynamic content shown on the landing page only.
type LandingData struct {
	LatestVersion   string
	Repo            GitHubRepo
	Git             GitStats
	RecentReleases  []GitHubRelease
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

// mdRenderer is the shared goldmark instance.
var mdRenderer = goldmark.New(
	goldmark.WithExtensions(extension.GFM, extension.Linkify, extension.TaskList, extension.Typographer),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(gmhtml.WithHardWraps(), gmhtml.WithUnsafe()),
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
)

// knownDocBases is the set of names that exist as <name>.html in the site.
// Used to rewrite bare-name links (no extension) emitted by the markdown
// source convention `[X](getting-started)` → works on GitHub, broken in
// static HTML.
var knownDocBases = map[string]bool{
	"getting-started":      true,
	"cli-reference":        true,
	"accounts-and-quota":   true,
	"coding-tools":         true,
	"library-guide":        true,
	"error-handling":       true,
	"architecture":         true,
	"roadmap":              true,
	"site-generation":      true,
	"CHANGELOG":            true,
	"CONTRIBUTING":         true,
	"SECURITY":             true,
	"CODE_OF_CONDUCT":      true,
}

// rewriteLinks takes goldmark-rendered HTML and rewrites internal markdown
// links to their static-HTML equivalents so links resolve in the generated
// site. GitHub renders `accounts-and-quota.md` as a working link; our static
// site needs `accounts-and-quota.html`.
//
// Three classes of rewrite:
//   1. `foo.md` (single name in same dir)  → `foo.html`
//   2. `getting-started` (bare known name) → `getting-started.html`
//   3. Any path with `../` segments or `docs/<lang>/...` prefix — these are
//      repo-tree-relative paths that don't match the site's URL layout. We
//      resolve them to the site-root-absolute URL of the target's basename.
//      Example: from inside docs/ru/architecture.md, `../../CONTRIBUTING.md`
//      → `/CONTRIBUTING.html`.
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

// RenderMarkdown converts markdown bytes to HTML. Used by both landing and doc
// rendering paths.
func RenderMarkdown(src []byte) (template.HTML, error) {
	var buf bytes.Buffer
	if err := mdRenderer.Convert(src, &buf); err != nil {
		return "", err
	}
	out := rewriteLinks(buf.Bytes())
	return template.HTML(out), nil
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
		"default":  func(d, v any) any { if isEmpty(v) { return d }; return v },
		"dict":     dict,
		"langBase": langBase,
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
	// The page template invokes {{ template "layout.html" . }} at the top;
	// layout.html then renders {{ block "content" . }} which is satisfied by
	// the {{ define "content" }}…{{ end }} block we just re-parsed.
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
