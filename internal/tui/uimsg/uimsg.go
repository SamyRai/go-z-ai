// Package uimsg holds tea.Msg types (and the Route helper) shared between the
// TUI root model and every screen subpackage, so screens can talk to the root
// without an import cycle (root imports every screen; screens must not import
// root).
package uimsg

import tea "charm.land/bubbletea/v2"

// Err is returned by a screen's tea.Cmd when an operation fails. The root
// model renders it as a status-line toast instead of crashing.
type Err struct{ Err error }

// Status carries a transient informational message for the status line.
type Status struct{ Text string }

// Routed carries a screen-specific message that must reach the screen that
// started the work, even if the user has since switched tabs. Async operations
// (e.g. the Media tab's video generation, which can run for minutes) wrap their
// terminal result in Routed so the root model delivers it to the originating
// screen instead of dropping it on whatever tab happens to be active when the
// work finishes. Tab is the destination screen's index; Msg is the wrapped,
// screen-private message (kept as any so uimsg needn't import bubbletea).
type Routed struct {
	Tab int
	Msg any
}

// Route wraps cmd so its result is delivered to screen tab as a Routed
// message, however the user moves between tabs while it runs.
func Route(tab int, cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg { return Routed{Tab: tab, Msg: cmd()} }
}

// CloseOverlay tells the root model to dismiss the currently-open overlay.
// Overlays (help, palette, model picker) emit this when the user presses esc,
// selects an item, or otherwise finishes — they can't clear themselves off
// the root's overlay slot without it, and they must not import the root
// package (the root imports them).
type CloseOverlay struct{}

// OpenModelPicker asks the root to open the chat model-picker overlay. The
// chat screen emits this on ctrl+o; the root builds the picker (it owns the
// client) and, on pick, forwards the chosen id back to chat. Splitting it
// this way keeps the picker as a root-owned overlay (consistent with help /
// palette) while the chat screen stays the source of truth for the model.
type OpenModelPicker struct{}

// AccountChanged tells the root that the active account changed (switched,
// added as the first account, or removed), so it can rebuild the API client
// every tab uses and refresh the header.
type AccountChanged struct{}

// PlanChanged tells the root that the stored GLM Coding Plan changed, so the
// header's plan badge can update without re-reading the store every frame.
type PlanChanged struct{ Plan string }
