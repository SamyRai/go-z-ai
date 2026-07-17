package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
)

// spyScreen is a minimal tea.Model that records the messages it receives, so a
// test can assert whether the root model delivered a message to it.
type spyScreen struct {
	got *[]tea.Msg
}

func (s spyScreen) Init() tea.Cmd { return nil }
func (s spyScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	*s.got = append(*s.got, msg)
	return s, nil
}
func (s spyScreen) View() tea.View { return tea.NewView("") }

// A uimsg.Routed message must be delivered to its target screen even when a
// different tab is active — this is what keeps an async result (e.g. a video
// generation that finishes after the user switched tabs) from being dropped.
func TestRoutedDeliversToInactiveScreen(t *testing.T) {
	// A minimal rootModel — newRootModel eagerly builds every real screen
	// (which needs live stores); the Routed dispatch only touches m.screens.
	m := &rootModel{active: tabChat}

	var received []tea.Msg
	m.screens[tabMedia] = spyScreen{got: &received}

	m.Update(uimsg.Routed{Tab: int(tabMedia), Msg: "ping"})

	if len(received) != 1 || received[0] != "ping" {
		t.Fatalf("expected the inactive target screen to receive \"ping\", got %v", received)
	}
}

// An out-of-range or nil-screen Routed tab is ignored rather than panicking.
func TestRoutedIgnoresBadTab(t *testing.T) {
	m := &rootModel{active: tabChat}
	// Should not panic: out of range low/high, and an in-range but nil screen.
	m.Update(uimsg.Routed{Tab: 999, Msg: "ping"})
	m.Update(uimsg.Routed{Tab: -1, Msg: "ping"})
	m.Update(uimsg.Routed{Tab: int(tabMedia), Msg: "ping"})
}
