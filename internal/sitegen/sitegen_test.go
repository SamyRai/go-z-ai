package sitegen

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestWriteSitemap_Deterministic is the regression for H5: two builds of the
// same input must produce byte-identical sitemap.xml. Previously the sitemap
// iterated localesWithFullDocs as a map (randomized order) and embedded
// time.Now() for <lastmod>.
func TestWriteSitemap_Deterministic(t *testing.T) {
	// Not t.Parallel — mutates package-level buildClock.
	clock := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	setBuildClock(clock)
	t.Cleanup(func() { setBuildClock(time.Time{}) })

	site := &SiteView{Owner: "SamyRai", Repo: "go-z-ai"}

	dir1 := t.TempDir()
	dir2 := t.TempDir()
	if err := writeSitemap(dir1, site); err != nil {
		t.Fatalf("writeSitemap dir1: %v", err)
	}
	if err := writeSitemap(dir2, site); err != nil {
		t.Fatalf("writeSitemap dir2: %v", err)
	}

	a, err := os.ReadFile(filepath.Join(dir1, "sitemap.xml"))
	if err != nil {
		t.Fatalf("read dir1: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir2, "sitemap.xml"))
	if err != nil {
		t.Fatalf("read dir2: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Errorf("sitemap non-deterministic across two builds of the same input\n--- build 1 ---\n%s\n--- build 2 ---\n%s", a, b)
	}

	// Spot-check expected content. The <lastmod> must be the build clock date,
	// not today's wall-clock date.
	want := "<lastmod>2026-07-19</lastmod>"
	if !strings.Contains(string(a), want) {
		t.Errorf("sitemap missing expected lastmod %q\noutput: %s", want, a)
	}
	// Locales within the doc-URL block must be in localesAll order: en, ru, zh.
	// We verify by checking that the en/ru/zh doc URLs for one canonical doc
	// appear in that order.
	enIdx := strings.Index(string(a), "/en/getting-started.html")
	ruIdx := strings.Index(string(a), "/ru/getting-started.html")
	zhIdx := strings.Index(string(a), "/zh/getting-started.html")
	for _, idx := range []int{enIdx, ruIdx, zhIdx} {
		if idx < 0 {
			t.Fatalf("sitemap missing one of /en/ru/zh/getting-started.html\noutput: %s", a)
		}
	}
	if enIdx >= ruIdx || ruIdx >= zhIdx {
		t.Errorf("locale doc URLs out of stable order: en=%d ru=%d zh=%d\noutput: %s", enIdx, ruIdx, zhIdx, a)
	}
}

// TestRun_RejectsBadIdentifiers covers M5: owner/repo/module are interpolated
// into URLs (github.com, api.github.com, pkg.go.dev, the sitemap host), so a
// value containing "/", "?", "#", or whitespace must be rejected up front
// rather than silently producing a malformed or wrong-host URL.
func TestRun_RejectsBadIdentifiers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		owner string
		repo  string
	}{
		{"owner with slash", "evil/path", "go-z-ai"},
		{"repo with question mark", "SamyRai", "go?zai"},
		{"owner with whitespace", "Samy Rai", "go-z-ai"},
		{"owner with hash", "SamyRai#", "go-z-ai"},
		{"empty owner", "", "go-z-ai"},
		{"empty repo", "SamyRai", ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := Run(context.Background(), Options{
				OutDir:      t.TempDir(),
				Owner:       tc.owner,
				Repo:        tc.repo,
				SkipNetwork: true,
				SourceDate:  time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
			})
			if err == nil {
				t.Errorf("expected error for owner=%q repo=%q, got nil", tc.owner, tc.repo)
			}
		})
	}
}

// TestRun_BadModule checks the module-path validator (allows "/" but not
// other URL-significant chars).
func TestRun_BadModule(t *testing.T) {
	t.Parallel()
	err := Run(context.Background(), Options{
		OutDir:      t.TempDir(),
		Owner:       "SamyRai",
		Repo:        "go-z-ai",
		Module:      "evil?query",
		SkipNetwork: true,
		SourceDate:  time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error for module with '?', got nil")
	}
}
