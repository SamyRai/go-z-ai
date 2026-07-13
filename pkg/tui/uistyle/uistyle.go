// Package uistyle holds the shared lipgloss style vocabulary used by the
// root chrome and every screen subpackage, so pill/border/toast colors stay
// consistent without pkg/tui's screens importing pkg/tui itself (which
// would create an import cycle, since pkg/tui imports every screen).
//
// Colors are built via lipgloss.Color only, never raw ANSI escapes, so
// Bubble Tea's colorprofile layer can auto-downsample them for
// NO_COLOR/16-color/dumb terminals with no extra fallback code required.
package uistyle

import "charm.land/lipgloss/v2"

var (
	ColorAccent   = lipgloss.Color("6")   // cyan
	ColorAccentBg = lipgloss.Color("23")  // dark teal, active-pill fill
	ColorMuted    = lipgloss.Color("8")   // bright black / gray
	ColorBorder   = lipgloss.Color("240") // dim gray panel border
	ColorError    = lipgloss.Color("1")   // red
	ColorWarn     = lipgloss.Color("3")   // yellow
	ColorSuccess  = lipgloss.Color("2")   // green

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
)

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
