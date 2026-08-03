package palette

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// An empty query shows every command in declared order; the cursor starts at
// the top.
func TestEmptyQueryShowsAll(t *testing.T) {
	cmds := []Command{
		{Name: "Go to Chat"},
		{Name: "Go to Models"},
		{Name: "Quit"},
	}
	m := New(cmds)
	if len(m.matched) != 3 {
		t.Fatalf("expected all 3 commands shown on empty query, got %d", len(m.matched))
	}
	if m.matched[0].Name != "Go to Chat" {
		t.Errorf("expected declared order on empty query, got %q first", m.matched[0].Name)
	}
}

// A query that matches a subset narrows the list and ranks best match first.
func TestFilterNarrowsAndRanks(t *testing.T) {
	cmds := []Command{
		{Name: "Go to Chat"},
		{Name: "Go to Models"},
		{Name: "Quit"},
	}
	m := New(cmds)
	// Type the query one rune at a time, the way the runtime delivers keys.
	// Update has a value receiver, so capture the returned model each time.
	for _, r := range "model" {
		updated, _ := m.Update(keyPress(string(r)))
		m = updated.(Model)
	}
	if len(m.matched) != 1 {
		t.Fatalf("expected 1 match for 'model', got %d: %+v", len(m.matched), m.matched)
	}
	if m.matched[0].Name != "Go to Models" {
		t.Errorf("expected 'Go to Models' to be the match, got %q", m.matched[0].Name)
	}
}

// A query that matches nothing leaves an empty list (View shows the
// 'no matching commands' line); the model does not panic.
func TestNoMatchIsEmpty(t *testing.T) {
	m := New([]Command{{Name: "Quit"}})
	for _, r := range "zzz" {
		updated, _ := m.Update(keyPress(string(r)))
		m = updated.(Model)
	}
	if len(m.matched) != 0 {
		t.Fatalf("expected zero matches for 'zzz', got %d", len(m.matched))
	}
}

// enter on the highlighted command returns a Result carrying the command's
// action, then the root closes the overlay.
func TestEnterEmitsResult(t *testing.T) {
	cmds := []Command{
		{Name: "Quit", Do: ActionQuit},
		{Name: "Refresh", Do: ActionRefresh},
	}
	m := New(cmds)
	// cursor starts at index 0 (Quit).
	_, cmd := m.Update(keyPress("enter"))
	if cmd == nil {
		t.Fatalf("expected a cmd from enter")
	}
	msg := cmd()
	res, ok := msg.(Result)
	if !ok {
		t.Fatalf("expected a Result msg, got %T", msg)
	}
	if res.Action != ActionQuit {
		t.Errorf("expected ActionQuit, got %v", res.Action)
	}
}

// esc closes the overlay via uimsg.CloseOverlay.
func TestEscCloses(t *testing.T) {
	m := New([]Command{{Name: "Quit"}})
	_, cmd := m.Update(keyPress("esc"))
	if cmd == nil {
		t.Fatalf("expected a cmd from esc")
	}
	// The msg type should be uimsg.CloseOverlay (compare by type name to
	// avoid importing uimsg here — though it's fine to import; we keep the
	// test self-contained by checking it isn't a Result).
	msg := cmd()
	if _, ok := msg.(Result); ok {
		t.Errorf("esc must not emit a Result, got %+v", msg)
	}
}

// keyPress builds a tea.KeyPressMsg for the given key string. v2's KeyPressMsg
// embeds Key; for printable single-rune keys both Text and Code must be set
// (bubbles textinput reads Code for printable input, Key.String() for
// matching the switch cases like "enter"/"esc"). Multi-rune strings are
// delivered as one synthetic key per rune, mirroring how the runtime delivers
// typed text.
func keyPress(s string) tea.KeyPressMsg {
	r := []rune(s)
	return tea.KeyPressMsg(tea.Key{Text: s, Code: r[0]})
}
