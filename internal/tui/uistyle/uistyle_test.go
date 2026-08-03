package uistyle

import (
	"strings"
	"testing"
)

// Each palette role must have distinct light/dark values, otherwise the
// "adaptive" theme is a no-op and we'd never notice.
func TestPaletteLightDarkDistinct(t *testing.T) {
	roles := []struct {
		name string
		c    lightDark
	}{
		{"accent", palette.accent},
		{"accentBg", palette.accentBg},
		{"muted", palette.muted},
		{"border", palette.border},
		{"err", palette.err},
		{"warn", palette.warn},
		{"success", palette.success},
	}
	for _, r := range roles {
		if r.c.Dark == r.c.Light {
			t.Errorf("role %s has identical dark/light values (%q); adaptive theme is a no-op", r.name, r.c.Dark)
		}
	}
}

// SetDark(true) then SetDark(false) must actually flip the resolved accent
// color, so a live theme switch (tea.BackgroundColorMsg) takes effect.
func TestSetDarkFlipsResolvedColors(t *testing.T) {
	original := isDark
	t.Cleanup(func() { SetDark(original) })

	SetDark(true)
	darkAccent := ColorAccent

	SetDark(false)
	lightAccent := ColorAccent

	if darkAccent == lightAccent {
		t.Fatalf("expected ColorAccent to differ between dark and light themes, both were %v", darkAccent)
	}
}

// SetDark is a no-op when the theme hasn't changed (avoids needless style
// rebuild churn on repeated BackgroundColorMsg).
func TestSetDarkNoOpWhenUnchanged(t *testing.T) {
	original := isDark
	t.Cleanup(func() { SetDark(original) })

	SetDark(true)
	before := ColorAccent
	SetDark(true) // same value — should not rebuild
	after := ColorAccent

	if before != after {
		t.Fatalf("expected ColorAccent unchanged on a no-op SetDark, got %v -> %v", before, after)
	}
}

// SkeletonRow renders exactly width block glyphs so the placeholder lines up
// with neighboring content. A non-positive width is clamped to 1.
func TestSkeletonRowWidth(t *testing.T) {
	for _, tc := range []struct {
		in   int
		want int
	}{
		{5, 5},
		{1, 1},
		{0, 1},  // clamped
		{-3, 1}, // clamped
		{40, 40},
	} {
		got := strings.Trim(SkeletonRow(tc.in), "\x1b[0123456789;m") // strip ANSI
		// Count only the block runes (the rendered string is just '▒' * width
		// wrapped in ANSI escapes from the Skeleton style).
		count := 0
		for _, r := range got {
			if r == '▒' {
				count++
			}
		}
		if count != tc.want {
			t.Errorf("SkeletonRow(%d): expected %d blocks, got %d", tc.in, tc.want, count)
		}
	}
}
