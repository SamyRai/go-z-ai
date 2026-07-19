package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
)

// helpOverlay is a modal that lists every keybinding available in the current
// screen plus the global ones. It's opened by the root model's "?" binding and
// self-closes on esc/?/q via uimsg.CloseOverlay.
type helpOverlay struct {
	sections []helpSection
}

// helpSection is one titled group of bindings inside the help card.
type helpSection struct {
	title    string
	bindings []key.Binding
}

// newHelpOverlay builds a help overlay from the global keymap plus the active
// screen's bindings. screenBindings is the result of helpProvider.ShortHelp
// for the active screen (may be empty).
func newHelpOverlay(global keyMap, screenTitle string, screenBindings []key.Binding) tea.Model {
	return &helpOverlay{
		sections: []helpSection{
			{title: "Global", bindings: []key.Binding{
				global.NextTab, global.PrevTab, global.Quit,
				global.Help, global.Palette,
			}},
			{title: screenTitle, bindings: screenBindings},
		},
	}
}

func (h *helpOverlay) Init() tea.Cmd { return nil }

func (h *helpOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return h, nil
	}
	switch kp.String() {
	case "esc", "?", "q", "enter":
		// Self-close: the overlay can't clear itself off the root's slot
		// without this signal (the root imports the overlay type, not vice
		// versa).
		return h, func() tea.Msg { return uimsg.CloseOverlay{} }
	}
	return h, nil
}

func (h *helpOverlay) View() tea.View {
	var b strings.Builder
	for _, sec := range h.sections {
		if len(sec.bindings) == 0 {
			continue
		}
		// Section header (the card title is added by renderOverlayCard; here
		// we render the per-section subtitle in plain text to keep widths
		// predictable).
		if sec.title != "" {
			b.WriteString(sec.title)
			b.WriteByte('\n')
		}
		// Two-column layout: keys (left, fixed 14) | help text (right). Keep
		// it width-stable so the card doesn't reflow on every binding.
		for _, kb := range sec.bindings {
			keys := kb.Help().Key
			desc := kb.Help().Desc
			if keys == "" {
				continue
			}
			pad := 14 - len(keys)
			if pad < 1 {
				pad = 1
			}
			b.WriteString(keys)
			b.WriteString(strings.Repeat(" ", pad))
			b.WriteString(desc)
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	body := strings.TrimRight(b.String(), "\n")
	return tea.NewView(renderOverlayCard("Keybindings", body))
}
