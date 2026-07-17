package tui

import "charm.land/bubbles/v2/key"

// keyMap holds the global key bindings handled by the root model before a
// keypress is ever delegated to the active screen.
type keyMap struct {
	Quit    key.Binding
	NextTab key.Binding
	PrevTab key.Binding
	Help    key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		// Quit is ctrl+c only: a bare "q" would fire while typing into any
		// screen's text input (chat message, prompts, API keys), since the
		// root model matches global keys before delegating.
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev tab"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
	}
}
