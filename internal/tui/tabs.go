package tui

import (
	"github.com/rivo/uniseg"

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

// pillWidth is the rendered cell width of a tab pill for the given name:
// PillActive/PillInactive both apply Padding(0, 2), i.e. 2 cells each side,
// around the name's display width.
func pillWidth(name string) int {
	w := 0
	rest := name
	for len(rest) > 0 {
		var cluster string
		cluster, rest, _, _ = uniseg.FirstGraphemeClusterInString(rest, -1)
		w += uniseg.StringWidth(cluster)
	}
	return w + 4 // 2 left + 2 right padding
}

// renderTabBar renders the horizontal tab strip with the active tab
// highlighted.
func renderTabBar(active tab) string {
	var bar string
	for i, name := range tabNames {
		if tab(i) == active {
			bar += uistyle.PillActive.Render(name)
		} else {
			bar += uistyle.PillInactive.Render(name)
		}
	}
	return bar
}

// tabBarHit returns the tab under the given x coordinate within the tab-bar
// row, and ok=false if x is past the last pill. Used by the root model to
// route a mouse click on the tab strip to the right tab.
func tabBarHit(x int) (tab, bool) {
	off := 0
	for i, name := range tabNames {
		w := pillWidth(name)
		if x >= off && x < off+w {
			return tab(i), true
		}
		off += w
	}
	return 0, false
}
