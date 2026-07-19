package sitegen

import (
	"strings"
	"testing"
	"time"
)

// TestRenderMarkdown_EscapesRawHTML is the regression test for the XSS sink
// that goldmark's WithUnsafe() used to open: a markdown source containing
// raw <script>/onerror markup must come out escaped/sanitized, not live HTML.
// Before the fix, RenderMarkdown returned the raw markup verbatim.
func TestRenderMarkdown_EscapesRawHTML(t *testing.T) {
	t.Parallel()
	src := []byte(`<script>alert('xss')</script>` + "\n\n" +
		`<img src=x onerror=alert(1)>` + "\n\n" +
		`[ok](https://example.com)` + "\n\nnormal text")

	out, err := RenderMarkdown(src)
	if err != nil {
		t.Fatalf("RenderMarkdown: %v", err)
	}
	got := string(out)

	for _, bad := range []string{
		"<script>",    // script tag must be stripped/escaped
		"onerror",     // event-handler attribute must be stripped
		"alert(",      // payload body must not survive in JS form
		"<iframe",     // no iframe passthrough
		"javascript:", // no javascript: URLs
	} {
		if strings.Contains(strings.ToLower(got), strings.ToLower(bad)) {
			t.Errorf("output contains forbidden substring %q\noutput: %s", bad, got)
		}
	}
	// Sanity: the legitimate link and prose survived.
	if !strings.Contains(got, "normal text") {
		t.Errorf("legitimate prose stripped by sanitizer\noutput: %s", got)
	}
	if !strings.Contains(got, `href="https://example.com"`) {
		t.Errorf("legitimate link stripped/rewritten by sanitizer\noutput: %s", got)
	}
}

// TestRenderMarkdown_LinkRewrite covers the .md → .html rewriter. The rewriter
// is the reason RenderMarkdown exists separately from goldmark — a regression
// here breaks every internal docs link on the static site.
func TestRenderMarkdown_LinkRewrite(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		md         string
		wantSubstr string
	}{
		{
			name:       "same-dir .md becomes .html",
			md:         "[getting-started](getting-started.md)",
			wantSubstr: `href="getting-started.html"`,
		},
		{
			name:       "bare known doc name becomes .html",
			md:         "[getting-started](getting-started)",
			wantSubstr: `href="getting-started.html"`,
		},
		{
			name:       "unknown bare name is left untouched",
			md:         "[x](some-unknown-page)",
			wantSubstr: `href="some-unknown-page"`,
		},
		{
			name:       "external URL is untouched",
			md:         "[pkg.go.dev](https://pkg.go.dev/example)",
			wantSubstr: `href="https://pkg.go.dev/example"`,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := RenderMarkdown([]byte(tc.md))
			if err != nil {
				t.Fatalf("RenderMarkdown: %v", err)
			}
			if !strings.Contains(string(out), tc.wantSubstr) {
				t.Errorf("want output to contain %q\noutput: %s", tc.wantSubstr, out)
			}
		})
	}
}

// TestRelativeTime_Deterministic verifies the build-clock fix: relativeTime
// computes "Xh ago" against buildClock (set via setBuildClock), not wall-clock
// time.Now. Two calls with a frozen build clock and the same input must agree.
func TestRelativeTime_Deterministic(t *testing.T) {
	// Not t.Parallel — mutates package-level buildClock.
	clock := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	setBuildClock(clock)
	t.Cleanup(func() { setBuildClock(time.Time{}) }) // restore wall-clock default

	commitTime := clock.Add(-3 * time.Hour)
	got := relativeTime(commitTime)
	if got != "3h ago" {
		t.Errorf("relativeTime: got %q, want %q", got, "3h ago")
	}

	// Two calls produce the same result regardless of when the test machine
	// runs them — the regression we're guarding against.
	got2 := relativeTime(commitTime)
	if got != got2 {
		t.Errorf("non-deterministic: %q vs %q", got, got2)
	}
}

// TestFullDocLocales_StableOrder guards the determinism fix: fullDocLocales
// must return the locales in the order they appear in localesAll, every call.
// Previously the code ranged the map directly, producing randomized order.
func TestFullDocLocales_StableOrder(t *testing.T) {
	t.Parallel()
	first := fullDocLocales()
	for i := 0; i < 20; i++ {
		got := fullDocLocales()
		if len(got) != len(first) {
			t.Fatalf("iter %d: len %d, want %d", i, len(got), len(first))
		}
		for j := range got {
			if got[j] != first[j] {
				t.Fatalf("iter %d: order differs at %d: got %v, want %v", i, j, got, first)
			}
		}
	}
	// Must be a subset of localesAll and in the same relative order.
	wantSet := map[string]bool{"en": true, "ru": true, "zh": true}
	if len(first) != len(wantSet) {
		t.Errorf("expected %d full-doc locales, got %d (%v)", len(wantSet), len(first), first)
	}
	for _, l := range first {
		if !wantSet[l] {
			t.Errorf("unexpected locale %q in %v", l, first)
		}
	}
}
