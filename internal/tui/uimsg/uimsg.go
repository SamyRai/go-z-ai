// Package uimsg holds tea.Msg types shared between the TUI root model and
// every screen subpackage. It has no dependency on pkg/tui or bubbletea's
// Model interface itself, only on tea.Msg's underlying type, so screens can
// report errors/status up to the root's status-line toast without an import
// cycle (root imports every screen; screens must not import root).
package uimsg

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
