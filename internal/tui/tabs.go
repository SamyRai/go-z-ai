package tui

import (
	"fmt"

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

// compactTabWidth is the terminal width below which the tab bar switches to a
// compact numbered form ("1:Chat 2:Models …") so it still fits one line
// without truncating the panel border or wrapping. The numbers are a visual
// aid only — tab/shift+tab still cycle (numeric 1-7 switching would clash
// with the Models screen's 1-4 filter keys, so it's intentionally not bound).
const compactTabWidth = 78

// renderTabBar renders the horizontal tab strip with the active tab
// highlighted. Below compactTabWidth it switches to a compact numbered form
// so the bar fits narrow terminals without wrapping over the panel.
func renderTabBar(active tab, width int) string {
	if width > 0 && width < compactTabWidth {
		return renderTabBarCompact(active)
	}
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

// renderTabBarCompact renders tabs as "N:Name" segments with a single space
// between, highlighting the active one with the accent color. Narrower than
// the padded pill form so it fits on ~60-col terminals.
func renderTabBarCompact(active tab) string {
	var bar string
	for i, name := range tabNames {
		seg := fmt.Sprintf("%d:%s", i+1, name)
		if tab(i) == active {
			seg = uistyle.Header.Render(seg) // accent + bold, no padding
		} else {
			seg = uistyle.PillInactive.Render(seg)
		}
		bar += seg + " "
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
