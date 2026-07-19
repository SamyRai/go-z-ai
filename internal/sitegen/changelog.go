package sitegen

import (
	"bufio"
	"html/template"
	"os"
	"regexp"
	"strings"
	"time"
)

// ChangelogRelease is one dated section of CHANGELOG.md.
type ChangelogRelease struct {
	Date      time.Time // parsed from "## YYYY-MM-DD" header; zero if unparseable
	DateLabel string    // raw header text (e.g. "2026-07-19" or "v0.1.0")
	Sections  []ChangelogSection
}

// ChangelogSection is a ### sub-heading within a release section.
type ChangelogSection struct {
	Name  string // "Added", "Changed", "Security", …
	Items []ChangelogItem
}

// ChangelogItem is one bullet.
type ChangelogItem struct {
	Raw  string // raw markdown text (without leading "- ")
	Text string // text with leading "- **name** —" stripped if present, for display
}

var (
	changelogHeaderRE = regexp.MustCompile(`^##\s+(\S.*)$`)
	changelogSubRE    = regexp.MustCompile(`^###\s+(\S.*)$`)
	changelogBulletRE = regexp.MustCompile(`^[-*]\s+(.*)$`)
	isoDateRE         = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})`)
)

// ParseChangelog reads CHANGELOG.md and returns dated sections, most recent first
// (as written). It stops at the first non-## content above the first release.
func ParseChangelog(path string) ([]ChangelogRelease, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var (
		releases []ChangelogRelease
		cur      *ChangelogRelease
		curSub   *ChangelogSection
		sc       = bufio.NewScanner(f)
	)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for sc.Scan() {
		line := sc.Text()

		// Top-of-file intro / other content before any "## " header.
		if h := matchGroup(changelogHeaderRE, line); h != "" {
			if cur != nil {
				flushSection(cur, curSub)
				releases = append(releases, *cur)
			}
			cur = &ChangelogRelease{DateLabel: strings.TrimSpace(h)}
			if m := isoDateRE.FindString(h); m != "" {
				if t, err := time.Parse("2006-01-02", m); err == nil {
					cur.Date = t
				}
			}
			curSub = nil
			continue
		}

		if cur == nil {
			continue // preamble before first release
		}

		if s := matchGroup(changelogSubRE, line); s != "" {
			flushSection(cur, curSub)
			curSub = &ChangelogSection{Name: strings.TrimSpace(s)}
			cur.Sections = append(cur.Sections, ChangelogSection{Name: curSub.Name})
			continue
		}

		if b := matchGroup(changelogBulletRE, line); b != "" {
			item := ChangelogItem{Raw: strings.TrimSpace(b)}
			item.Text = simplifyBullet(item.Raw)
			if curSub != nil {
				curSub.Items = append(curSub.Items, item)
			} else {
				// Bullets with no enclosing ### section — put in a default section.
				if len(cur.Sections) == 0 || cur.Sections[len(cur.Sections)-1].Name != "" {
					cur.Sections = append(cur.Sections, ChangelogSection{Name: ""})
				}
				cur.Sections[len(cur.Sections)-1].Items =
					append(cur.Sections[len(cur.Sections)-1].Items, item)
			}
			continue
		}
	}
	if sc.Err() != nil {
		return nil, sc.Err()
	}
	if cur != nil {
		flushSection(cur, curSub)
		releases = append(releases, *cur)
	}
	return releases, nil
}

// flushSection finalises curSub by appending it to cur if non-empty.
func flushSection(cur *ChangelogRelease, sub *ChangelogSection) {
	if sub == nil || len(sub.Items) == 0 {
		return
	}
	// Replace the placeholder section we appended when curSub was created.
	for i := range cur.Sections {
		if cur.Sections[i].Name == sub.Name && len(cur.Sections[i].Items) == 0 {
			cur.Sections[i].Items = sub.Items
			return
		}
	}
}

var bulletLeadRE = regexp.MustCompile(`^\*\*([^*]+)\*\*\s*[—–-]\s*(.*)$`)

// simplifyBullet strips leading "**Name** —" from bullets like "- **chat.go** — added foo".
// The captured groups are HTML-escaped — this string is later interpolated
// into a template as HTML markup, so untrusted CHANGELOG content must not
// be concatenated in raw.
func simplifyBullet(raw string) string {
	if m := bulletLeadRE.FindStringSubmatch(raw); m != nil {
		return "<strong>" + htmlEsc(m[1]) + "</strong> — " + htmlEsc(m[2])
	}
	return htmlEsc(raw)
}

// htmlEsc is a short alias for template.HTMLEscapeString to keep the call
// sites in this file readable.
func htmlEsc(s string) string { return template.HTMLEscapeString(s) }

func matchGroup(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return m[1]
}
