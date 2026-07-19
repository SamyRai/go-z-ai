// Package uistyle holds the shared lipgloss style vocabulary used by the
// root chrome and every screen subpackage, so pill/border/toast colors stay
// consistent without pkg/tui's screens importing pkg/tui itself (which
// would create an import cycle, since pkg/tui imports every screen).
//
// Theme. The palette is dual (light + dark pairs) and resolved at render
// time against the terminal's current background. The root model calls
// SetDark once it has a tea.BackgroundColorMsg (defaulting to dark until
// then, matching what most developers run); the resolved styles are package
// vars, reassigned by applyTheme, so every View() that reads them picks up
// the new theme on the next frame.
//
// Colors are built via lipgloss.Color only, never raw ANSI escapes, so
// Bubble Tea's colorprofile layer can auto-downsample them for
// NO_COLOR/16-color/dumb terminals with no extra fallback code required.
package uistyle

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// palette holds the light/dark color pair for each role. Each role is a
// 256-color code (string) so it downsamples cleanly on limited terminals.
var palette = struct {
	accent   lightDark
	accentBg lightDark
	muted    lightDark
	border   lightDark
	err      lightDark
	warn     lightDark
	success  lightDark
}{
	accent:   lightDark{Dark: "6", Light: "30"},   // cyan
	accentBg: lightDark{Dark: "23", Light: "153"}, // dark teal / pale cyan
	muted:    lightDark{Dark: "8", Light: "245"},  // gray
	border:   lightDark{Dark: "240", Light: "250"},
	err:      lightDark{Dark: "1", Light: "160"},
	warn:     lightDark{Dark: "3", Light: "136"},
	success:  lightDark{Dark: "2", Light: "34"},
}

type lightDark struct{ Dark, Light string }

// isDark tracks the terminal's resolved background. Defaults to true (the
// common case); updated by SetDark when the root model receives a
// tea.BackgroundColorMsg.
var isDark = true

// SetDark reconfigures the palette against the terminal's background and
// rebuilds every exported style so subsequent renders pick up the new theme.
// Safe to call from any goroutine; called from the Bubble Tea Update loop.
func SetDark(dark bool) {
	if dark == isDark {
		return
	}
	isDark = dark
	applyTheme()
}

// IsDark reports the currently-resolved theme. Screens that cache
// theme-dependent state (e.g. chat's glamour renderer) use this to decide
// when to rebuild.
func IsDark() bool { return isDark }

// Resolved color roles (read by callers at render time). Reassigned by
// applyTheme on SetDark.
var (
	ColorAccent   color.Color = lipgloss.Color(palette.accent.Dark)
	ColorAccentBg color.Color = lipgloss.Color(palette.accentBg.Dark)
	ColorMuted    color.Color = lipgloss.Color(palette.muted.Dark)
	ColorBorder   color.Color = lipgloss.Color(palette.border.Dark)
	ColorError    color.Color = lipgloss.Color(palette.err.Dark)
	ColorWarn     color.Color = lipgloss.Color(palette.warn.Dark)
	ColorSuccess  color.Color = lipgloss.Color(palette.success.Dark)
)

// Styles (read by callers at render time). Reassigned by applyTheme.
var (
	// PillActive/PillInactive render a filled rounded "pill" segment, used
	// for both the root tab bar and in-screen filter rows (Models tab).
	PillActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(ColorAccentBg).
			Padding(0, 2)

	PillInactive = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 2)

	Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent).
		Padding(0, 1)

	// Panel wraps a screen's content in a bordered container. Only the root
	// model applies this around the active screen — screens themselves
	// should not nest another Panel border inside their own View, or the
	// app ends up with double-boxed content.
	Panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	StatusBar = lipgloss.NewStyle().Foreground(ColorMuted)

	ToastError = lipgloss.NewStyle().Bold(true).Foreground(ColorError)
	ToastWarn  = lipgloss.NewStyle().Foreground(ColorWarn)
	ToastInfo  = lipgloss.NewStyle().Foreground(ColorSuccess)

	// SectionTitle labels a sub-panel within a screen (e.g. "Model token
	// usage" above the Usage tab's heatmap).
	SectionTitle = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)

	// Subtle renders secondary/supporting text (e.g. the quota burn-rate hint)
	// in muted gray so it reads as annotation, not primary data.
	Subtle = lipgloss.NewStyle().Foreground(ColorMuted)

	// Skeleton renders placeholder block rows while async data is loading, so
	// the layout reads as "filling in" instead of "empty then jump".
	Skeleton = lipgloss.NewStyle().Foreground(ColorMuted)

	// EmptyTitle / EmptyHint render friendly empty-state messages with a
	// one-line call-to-action.
	EmptyTitle = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	EmptyHint  = lipgloss.NewStyle().Foreground(ColorMuted)
)

// pick returns the active-palette color for a role under the current theme.
func pick(c lightDark) color.Color {
	if isDark {
		return lipgloss.Color(c.Dark)
	}
	return lipgloss.Color(c.Light)
}

// applyTheme reassigns the color roles and rebuilds every exported style for
// the current isDark value. Called once on init and again whenever SetDark
// flips the theme.
func applyTheme() {
	ColorAccent = pick(palette.accent)
	ColorAccentBg = pick(palette.accentBg)
	ColorMuted = pick(palette.muted)
	ColorBorder = pick(palette.border)
	ColorError = pick(palette.err)
	ColorWarn = pick(palette.warn)
	ColorSuccess = pick(palette.success)

	PillActive = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(ColorAccentBg).
		Padding(0, 2)
	PillInactive = lipgloss.NewStyle().Foreground(ColorMuted).Padding(0, 2)
	Header = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent).Padding(0, 1)
	Panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)
	StatusBar = lipgloss.NewStyle().Foreground(ColorMuted)
	ToastError = lipgloss.NewStyle().Bold(true).Foreground(ColorError)
	ToastWarn = lipgloss.NewStyle().Foreground(ColorWarn)
	ToastInfo = lipgloss.NewStyle().Foreground(ColorSuccess)
	SectionTitle = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	Subtle = lipgloss.NewStyle().Foreground(ColorMuted)
	Skeleton = lipgloss.NewStyle().Foreground(ColorMuted)
	EmptyTitle = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	EmptyHint = lipgloss.NewStyle().Foreground(ColorMuted)
}

func init() { applyTheme() }

// RenderPills renders names as a row of pill segments, highlighting active.
func RenderPills(active int, names []string) string {
	var out string
	for i, name := range names {
		if i == active {
			out += PillActive.Render(name)
		} else {
			out += PillInactive.Render(name)
		}
	}
	return out
}

// SkeletonRow renders a dimmed block row of the given cell width, used as a
// placeholder while a row of real content is loading. width is clamped to >=1.
func SkeletonRow(width int) string {
	if width < 1 {
		width = 1
	}
	row := make([]rune, width)
	for i := range row {
		row[i] = '▒'
	}
	return Skeleton.Render(string(row))
}
