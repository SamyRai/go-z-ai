package tools

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func sized(m Model) Model {
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return next.(Model)
}

// A result clears busy and renders; an error renders as an error line.
func TestResultAndError(t *testing.T) {
	m := sized(New(nil))
	m.busy = true

	next, _ := m.Update(resultMsg{text: "hits"})
	got := next.(Model)
	if got.busy {
		t.Error("expected busy cleared on result")
	}
	if !strings.Contains(got.View().Content, "hits") {
		t.Errorf("expected result text, got:\n%s", got.View().Content)
	}

	errored, _ := sized(New(nil)).Update(resultMsg{err: errors.New("nope")})
	if !strings.Contains(errored.(Model).View().Content, "error: nope") {
		t.Error("expected error line in view")
	}
}

// up/down cycle the active form.
func TestFormCycle(t *testing.T) {
	m := New(nil)
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
	m := New(nil)
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
