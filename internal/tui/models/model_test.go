package models

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

func keyRune(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

// A successful fetch clears loading and populates the table filter set.
func TestFetchedPopulates(t *testing.T) {
	m := New(nil)
	m.loading = true

	next, _ := m.Update(fetchedMsg{models: []client.ModelDetails{
		{ID: "glm-4.6", ContextSize: 128000, OwnedBy: "z.ai"},
		{ID: "glm-4.6v", ContextSize: 64000, OwnedBy: "z.ai"},
	}})
	got := next.(Model)
	if got.loading {
		t.Error("expected loading cleared after fetch")
	}
	if len(got.all) != 2 {
		t.Errorf("expected 2 models stored, got %d", len(got.all))
	}
}

// A fetch error is surfaced to the root as a uimsg.Err toast, not a crash.
func TestFetchedErrorRaisesToast(t *testing.T) {
	m := New(nil)
	m.loading = true

	next, cmd := m.Update(fetchedMsg{err: errors.New("boom")})
	if next.(Model).loading {
		t.Error("expected loading cleared even on error")
	}
	if cmd == nil {
		t.Fatal("expected a command emitting uimsg.Err")
	}
	if _, ok := cmd().(uimsg.Err); !ok {
		t.Error("expected the error to be raised as uimsg.Err")
	}
}

// Number keys switch the active filter.
func TestFilterKeys(t *testing.T) {
	m := New(nil)
	for key, want := range map[rune]filter{'2': filterText, '3': filterVision, '4': filterFree, '1': filterAll} {
		next, _ := m.Update(keyRune(key))
		if got := next.(Model).filter; got != want {
			t.Errorf("key %q: filter = %d, want %d", key, got, want)
		}
	}
}

// 'r' triggers a reload (loading + a fetch command).
func TestReloadKey(t *testing.T) {
	m := New(nil)
	next, cmd := m.Update(keyRune('r'))
	if !next.(Model).loading {
		t.Error("expected loading set on reload")
	}
	if cmd == nil {
		t.Error("expected a fetch command on reload")
	}
}
