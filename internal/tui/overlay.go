package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/modelpicker"
	"github.com/SamyRai/go-z-ai/internal/tui/palette"
	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
)

// Overlays (help, command palette, model picker) are modal tea.Models the
// root composites over the active screen; this file builds them and carries
// out palette actions.

// placeOverlay centers content over a blank backdrop of width×height, so the
// screen beneath is obscured rather than painted over.
func placeOverlay(width, height int, content string) string {
	if width < 1 || height < 1 {
		return content
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content,
		lipgloss.WithWhitespaceChars(" "))
}

// openHelpOverlay builds the help overlay from the global keymap and the
// active screen's bindings.
func (m *rootModel) openHelpOverlay() tea.Model {
	var screenBindings []key.Binding
	if h, ok := m.screens[m.active].(helpProvider); ok {
		screenBindings = h.ShortHelp()
	}
	return newHelpOverlay(m.keys, tabNames[m.active], screenBindings)
}

// openModelPicker builds the chat model picker, highlighting the chat
// screen's current model. It fetches the catalog on Init.
func (m *rootModel) openModelPicker() tea.Model {
	current := ""
	if g, ok := m.screens[tabChat].(chatModelGetter); ok {
		current = g.ModelID()
	}
	return modelpicker.New(m.session.Client, current)
}

// openPalette builds the command palette.
func (m *rootModel) openPalette() tea.Model {
	cmds := make([]palette.Command, 0, len(tabNames)+4)
	for i, name := range tabNames {
		cmds = append(cmds, palette.Command{Name: "Go to " + name, Desc: "switch tab", Do: palette.ActionSwitchTab, DoArg: i})
	}
	cmds = append(cmds,
		palette.Command{Name: "Refresh current tab", Desc: "reload data", Do: palette.ActionRefresh},
		palette.Command{Name: "Toggle help", Desc: "open the keybindings overlay", Do: palette.ActionToggleHelp},
		palette.Command{Name: "Switch chat model", Desc: "open the model picker (chat tab)", Do: palette.ActionOpenModelPicker},
		palette.Command{Name: "Quit", Desc: "exit go-z-ai tui", Do: palette.ActionQuit},
	)
	return palette.New(cmds)
}

// runPaletteAction performs the action chosen in the palette, which the
// caller has already dismissed.
func (m *rootModel) runPaletteAction(res palette.Result) tea.Cmd {
	switch res.Action {
	case palette.ActionSwitchTab:
		if res.Arg >= 0 && res.Arg < int(tabCount) && m.screens[res.Arg] != nil {
			m.switchTab(tab(res.Arg))
			return m.ensureInit()
		}
	case palette.ActionRefresh:
		return m.refreshScreen(m.active)
	case palette.ActionToggleHelp:
		m.overlay = m.openHelpOverlay()
	case palette.ActionOpenModelPicker:
		openPicker := func() tea.Msg { return uimsg.OpenModelPicker{} }
		if m.active != tabChat && m.screens[tabChat] != nil {
			m.switchTab(tabChat)
			return tea.Batch(m.ensureInit(), openPicker)
		}
		return openPicker
	case palette.ActionQuit:
		// The streaming guard applies here too: the palette must not be a
		// back door around it.
		if !m.activeStreaming() {
			return tea.Quit
		}
	}
	return nil
}
