// Package uistyle holds the shared lipgloss style vocabulary used by the
// root chrome and every screen subpackage, so pill/border/toast colors stay
// consistent without pkg/tui's screens importing pkg/tui itself (which
// would create an import cycle, since pkg/tui imports every screen).
//
// Theme. The palette is dual (light + dark pairs) and resolved at render
// time against the terminal's current background. The root model calls
// SetDark once it has a tea.BackgroundColorMsg (defaulting to dark until
// then, matching what most developers run); the styles are package vars
// rebuilt by applyTheme, so every View() picks up the new theme on the next
// frame.
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
// rebuilds every style so subsequent renders pick up the new theme. Called
// from the Bubble Tea Update loop (not goroutine-safe).
func SetDark(dark bool) {
	if dark == isDark {
		return
	}
	isDark = dark
	applyTheme()
}

// Resolved color roles; set by applyTheme.
var (
	colorAccent   color.Color
	colorAccentBg color.Color
	colorMuted    color.Color
	colorBorder   color.Color
	colorError    color.Color
	colorWarn     color.Color
	colorSuccess  color.Color
)

// Styles, read by callers at render time; built by applyTheme.
var (
	// Chip is white text on the accent background — the base of the header
	// app badge, the active pill, and inline tags.
	Chip lipgloss.Style
	// PillActive/PillInactive render the segments of the root tab bar and
	// in-screen filter rows.
	PillActive   lipgloss.Style
	PillInactive lipgloss.Style
	// Header titles a screen; HeaderApp is the header's app-name badge.
	Header    lipgloss.Style
	HeaderApp lipgloss.Style
	// BadgeValue/BadgeWarn/BadgeOK color a header chip's value (RenderBadge).
	BadgeValue lipgloss.Style
	BadgeWarn  lipgloss.Style
	BadgeOK    lipgloss.Style
	// Panel wraps the active screen. Only the root applies it — a screen that
	// adds its own border renders double-boxed.
	Panel lipgloss.Style
	// Toast styles color the status line by severity.
	ToastError lipgloss.Style
	ToastWarn  lipgloss.Style
	ToastInfo  lipgloss.Style
	// SectionTitle labels a sub-panel; Subtle renders secondary text.
	SectionTitle lipgloss.Style
	Subtle       lipgloss.Style
	// EmptyTitle/EmptyHint render an empty state and its call to action.
	EmptyTitle lipgloss.Style
	EmptyHint  lipgloss.Style

	badgeLabel  lipgloss.Style
	skeleton    lipgloss.Style
	overlayCard lipgloss.Style
)

// pick returns the active-palette color for a role under the current theme.
func pick(c lightDark) color.Color {
	if isDark {
		return lipgloss.Color(c.Dark)
	}
	return lipgloss.Color(c.Light)
}

// applyTheme resolves the color roles and builds every style for the current
// theme — the one place styles are defined. Called on init and whenever
// SetDark flips the theme.
func applyTheme() {
	colorAccent = pick(palette.accent)
	colorAccentBg = pick(palette.accentBg)
	colorMuted = pick(palette.muted)
	colorBorder = pick(palette.border)
	colorError = pick(palette.err)
	colorWarn = pick(palette.warn)
	colorSuccess = pick(palette.success)

	muted := lipgloss.NewStyle().Foreground(colorMuted)
	accent := lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	chip := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(colorAccentBg)
	rounded := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())

	Chip = chip.Padding(0, 1)
	PillActive = chip.Bold(true).Padding(0, 2)
	PillInactive = muted.Padding(0, 2)
	Header = accent.Padding(0, 1)
	HeaderApp = chip.Bold(true).Padding(0, 1)
	BadgeValue = accent
	BadgeWarn = lipgloss.NewStyle().Bold(true).Foreground(colorWarn)
	BadgeOK = lipgloss.NewStyle().Bold(true).Foreground(colorSuccess)
	Panel = rounded.BorderForeground(colorBorder).Padding(0, 1)
	ToastError = lipgloss.NewStyle().Bold(true).Foreground(colorError)
	ToastWarn = lipgloss.NewStyle().Foreground(colorWarn)
	ToastInfo = lipgloss.NewStyle().Foreground(colorSuccess)
	SectionTitle = accent
	Subtle = muted
	EmptyTitle = accent
	EmptyHint = muted
	badgeLabel = muted
	skeleton = muted
	// The overlay card reuses the panel's border shape so a modal reads as a
	// panel on top of the panel.
	overlayCard = rounded.BorderForeground(colorAccent).Padding(1, 2)
}

func init() { applyTheme() }

// RenderOverlayCard wraps a titled body in the modal card style. Shared by
// the root's help overlay and subpackage overlays (palette, model picker).
func RenderOverlayCard(title, body string) string {
	card := body
	if title != "" {
		card = SectionTitle.Render(title) + "\n\n" + body
	}
	return overlayCard.Render(card)
}

// RenderBadge renders a "label value" chip: a muted label followed by a
// styled value, with a single space between. Used by the top header's
// account / plan / model context chips. valueStyle is one of BadgeValue /
// BadgeWarn / BadgeOK (or any lipgloss style the caller supplies).
func RenderBadge(label, value string, valueStyle lipgloss.Style) string {
	if value == "" {
		return ""
	}
	return badgeLabel.Render(label+" ") + valueStyle.Render(value)
}

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
	return skeleton.Render(string(row))
}
