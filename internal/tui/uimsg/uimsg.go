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
