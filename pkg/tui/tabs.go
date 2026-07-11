package tui

import "zai-api-client/pkg/tui/uistyle"

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
