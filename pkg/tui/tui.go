// Package tui implements the zai-client interactive terminal UI: a Bubble
// Tea v2 program with one tab per existing CLI command group, all wired to
// the same pkg/client, pkg/accounts, and pkg/coding services the
// non-interactive commands already use.
package tui

import (
	tea "charm.land/bubbletea/v2"
)

// Run builds and runs the TUI program until the user quits. Bubble Tea v2
// catches panics by default (see tea.WithoutCatchPanics) and restores the
// terminal on exit, so callers just need to propagate the returned error.
func Run(cfg Config) error {
	p := tea.NewProgram(newRootModel(cfg))
	_, err := p.Run()
	return err
}
