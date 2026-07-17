package tools

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
)

// route addresses a tab's result to itself so it survives a tab switch.
func TestRouteWrapsToSelfTab(t *testing.T) {
	m := New(nil, 3)
	msg := m.route(func() tea.Msg { return resultMsg{text: "ok"} })()
	routed, ok := msg.(uimsg.Routed)
	if !ok {
		t.Fatalf("expected uimsg.Routed, got %T", msg)
	}
	if routed.Tab != 3 {
		t.Errorf("expected result routed to tab 3, got %d", routed.Tab)
	}
	if _, ok := routed.Msg.(resultMsg); !ok {
		t.Errorf("expected wrapped resultMsg, got %T", routed.Msg)
	}
}

func sized(m Model) Model {
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return next.(Model)
}

// A result clears busy and renders; an error renders as an error line.
func TestResultAndError(t *testing.T) {
	m := sized(New(nil, 5))
	m.busy = true

	next, _ := m.Update(resultMsg{text: "hits"})
	got := next.(Model)
	if got.busy {
		t.Error("expected busy cleared on result")
	}
	if !strings.Contains(got.View().Content, "hits") {
		t.Errorf("expected result text, got:\n%s", got.View().Content)
	}

	errored, _ := sized(New(nil, 5)).Update(resultMsg{err: errors.New("nope")})
	if !strings.Contains(errored.(Model).View().Content, "error: nope") {
		t.Error("expected error line in view")
	}
}

// up/down cycle the active form.
func TestFormCycle(t *testing.T) {
	m := New(nil, 5)
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if next.(Model).active != formReader {
		t.Errorf("down should advance to formReader, got %d", next.(Model).active)
	}
	back, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if back.(Model).active != formTokenize {
		t.Errorf("up should wrap to formTokenize, got %d", back.(Model).active)
	}
}

// Enter starts work (busy + command); enter while busy is a no-op.
func TestEnterBusyGuard(t *testing.T) {
	m := New(nil, 5)
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !next.(Model).busy || cmd == nil {
		t.Error("expected enter to set busy and return a command")
	}

	busy := next.(Model)
	_, cmd2 := busy.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd2 != nil {
		t.Error("expected enter while busy to be a no-op")
	}
}
