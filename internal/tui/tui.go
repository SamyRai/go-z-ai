// Package tui implements the go-z-ai interactive terminal UI: a Bubble Tea v2
// program with one tab per CLI command group, wired to the same pkg/client,
// internal/accounts, and internal/coding services the commands use.
package tui

import (
	tea "charm.land/bubbletea/v2"
)

// Run builds and runs the TUI program until the user quits. Bubble Tea v2
// catches panics by default (see tea.WithoutCatchPanics) and restores the
// terminal on exit, so callers just need to propagate the returned error.
func Run(cfg Config) error {
	s, err := newSession(cfg)
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(newRootModel(s)).Run()
	return err
}
