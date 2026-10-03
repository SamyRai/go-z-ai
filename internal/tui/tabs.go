package tui

import (
	"charm.land/lipgloss/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
)

// tab identifies one of the top-level screens.
type tab int

const (
	tabChat tab = iota
	tabModels
	tabUsage
	tabAccounts
	tabCoding
	tabMedia
	tabTools
	tabCount
)

var tabNames = [tabCount]string{
	tabChat:     "Chat",
	tabModels:   "Models",
	tabUsage:    "Usage",
	tabAccounts: "Accounts",
	tabCoding:   "Coding",
	tabMedia:    "Media",
	tabTools:    "Tools",
}

// tabSegments renders each tab's segment for the given terminal width: padded
// pills when the full bar fits, otherwise bare names separated by single
// spaces. Rendering and mouse hit-testing both use this,
// so a click always lands on the tab drawn under it.
func tabSegments(active tab, width int) []string {
	compact := width > 0 && width < fullTabBarWidth()
	segs := make([]string, tabCount)
	for i, name := range tabNames {
		isActive := tab(i) == active
		switch {
		case !compact && isActive:
			segs[i] = uistyle.PillActive.Render(name)
		case !compact:
			segs[i] = uistyle.PillInactive.Render(name)
		case isActive:
			segs[i] = uistyle.Header.Render(name)
		default:
			segs[i] = uistyle.Subtle.Render(name)
		}
		if compact && i < len(tabNames)-1 {
			segs[i] += " "
		}
	}
	return segs
}

// fullTabBarWidth is the width of the padded-pill tab bar.
func fullTabBarWidth() int {
	w := 0
	for i := range tabNames {
		w += lipgloss.Width(uistyle.PillInactive.Render(tabNames[i]))
	}
	return w
}

// renderTabBar renders the tab strip with the active tab highlighted.
func renderTabBar(active tab, width int) string {
	var bar string
	for _, s := range tabSegments(active, width) {
		bar += s
	}
	return bar
}

// tabBarHit returns the tab drawn at column x of the tab bar.
func tabBarHit(x int, active tab, width int) (tab, bool) {
	off := 0
	for i, s := range tabSegments(active, width) {
		w := lipgloss.Width(s)
		if x >= off && x < off+w {
			return tab(i), true
		}
		off += w
	}
	return 0, false
}
