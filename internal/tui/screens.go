package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Optional capabilities a screen can implement; the root probes for them.

// streamer is implemented by screens that must keep focus while an operation
// runs (e.g. a chat stream, whose chunks are delivered to the active screen).
type streamer interface {
	Streaming() bool
}

// inputCapturer is implemented by screens with a text field that currently
// has focus; while it does, printable global keys (like "?") are typed into
// the field instead of triggering the root's shortcut.
type inputCapturer interface {
	CapturesInput() bool
}

// helpProvider is implemented by screens whose own key bindings belong in
// the footer and help overlay.
type helpProvider interface {
	ShortHelp() []key.Binding
}

// refresher is implemented by screens that can reload their data (the 'r'
// key, the palette's "Refresh current tab", and account switches).
type refresher interface {
	Refresh() (tea.Model, tea.Cmd)
}

// chatModelSetter / chatModelGetter let the root move the chat model between
// the model picker and the chat screen, which stays its source of truth.
type chatModelSetter interface {
	SetModel(id string) (tea.Model, tea.Cmd)
}

type chatModelGetter interface {
	ModelID() string
}

// activeStreaming reports whether the active screen has an operation that
// pins it in place.
func (m *rootModel) activeStreaming() bool {
	s, ok := m.screens[m.active].(streamer)
	return ok && s.Streaming()
}

// activeCapturesInput reports whether the active screen's text field has
// focus.
func (m *rootModel) activeCapturesInput() bool {
	s, ok := m.screens[m.active].(inputCapturer)
	return ok && s.CapturesInput()
}
