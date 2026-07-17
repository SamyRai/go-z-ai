package coding

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
)

// A refresh error is raised as a toast rather than crashing the tab.
func TestCodingRefreshErrorRaisesToast(t *testing.T) {
	m := New(nil)
	_, cmd := m.Update(refreshedMsg{err: errors.New("boom")})
	if cmd == nil {
		t.Fatal("expected a command on refresh error")
	}
	if _, ok := cmd().(uimsg.Err); !ok {
		t.Error("expected refreshedMsg error to raise uimsg.Err")
	}
}

// A completed action with no error triggers a refresh; with an error it toasts.
func TestCodingActionDone(t *testing.T) {
	m := New(nil)
	if _, cmd := m.Update(actionDoneMsg{}); cmd == nil {
		t.Error("expected a refresh command after a successful action")
	}

	_, errCmd := m.Update(actionDoneMsg{err: errors.New("nope")})
	if errCmd == nil {
		t.Fatal("expected a command on action error")
	}
	if _, isErr := errCmd().(uimsg.Err); !isErr {
		t.Error("expected uimsg.Err on action error")
	}
}

// 'a' opens the auth form (mode switches, a form + init command appear).
func TestCodingAuthKeyOpensForm(t *testing.T) {
	m := New(nil)
	next, cmd := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	got := next.(Model)
	if got.mode != modeAuth {
		t.Error("expected mode switched to auth")
	}
	if got.form == nil || cmd == nil {
		t.Error("expected an auth form and its init command")
	}
}
