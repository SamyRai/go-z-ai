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
	m := New(nil, 5)
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
	m := New(nil, 5)
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
	m := New(nil, 5)
	for key, want := range map[rune]filter{'2': filterText, '3': filterVision, '4': filterFree, '1': filterAll} {
		next, _ := m.Update(keyRune(key))
		if got := next.(Model).filter; got != want {
			t.Errorf("key %q: filter = %d, want %d", key, got, want)
		}
	}
}

// route addresses the fetch result to this tab so it survives a tab switch.
func TestRouteWrapsToSelfTab(t *testing.T) {
	m := New(nil, 3)
	msg := m.route(func() tea.Msg { return fetchedMsg{} })()
	routed, ok := msg.(uimsg.Routed)
	if !ok {
		t.Fatalf("expected uimsg.Routed, got %T", msg)
	}
	if routed.Tab != 3 {
		t.Errorf("expected fetch result routed to tab 3, got %d", routed.Tab)
	}
	if _, ok := routed.Msg.(fetchedMsg); !ok {
		t.Errorf("expected wrapped fetchedMsg, got %T", routed.Msg)
	}
}

// 'r' triggers a reload (loading + a fetch command).
func TestReloadKey(t *testing.T) {
	m := New(nil, 5)
	next, cmd := m.Update(keyRune('r'))
	if !next.(Model).loading {
		t.Error("expected loading set on reload")
	}
	if cmd == nil {
		t.Error("expected a fetch command on reload")
	}
}

// Enter opens the full-screen detail view for the highlighted row; Esc
// returns to the table. State-machine behavior, mirroring Accounts' mode enum.
func TestEnterOpensDetailEscReturns(t *testing.T) {
	m := New(nil, 5)
	m.all = []client.ModelDetails{
		{ID: "glm-4.6", OwnedBy: "z-ai"},
	}
	m.applyFilter()

	if m.mode != modeTable {
		t.Fatalf("expected to start in modeTable, got %d", m.mode)
	}
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if got := next.(Model).mode; got != modeDetail {
		t.Errorf("enter: expected modeDetail, got %d", got)
	}
	// In detail mode, ShortHelp should advertise the esc binding.
	detailHelp := next.(Model).ShortHelp()
	if len(detailHelp) == 0 || detailHelp[0].Help().Key != "esc" {
		t.Errorf("detail ShortHelp should start with esc, got %+v", detailHelp)
	}
	next, _ = next.(Model).Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if got := next.(Model).mode; got != modeTable {
		t.Errorf("esc: expected modeTable, got %d", got)
	}
}

// Enter with no rows selected is a no-op (no crash, stays in table mode).
func TestEnterNoSelectionIsSafe(t *testing.T) {
	m := New(nil, 5) // no models loaded
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if got := next.(Model).mode; got != modeTable {
		t.Errorf("enter with no rows should stay in table mode, got %d", got)
	}
}

// The Text filter excludes vision models (users picking "Text" want the
// pure-text subset), and the Vision filter includes only vision-capable
// models. Both now read from the single catalog source of truth, so this also
// guards against the heuristics drifting apart again.
func TestTextAndVisionFiltersUseCapabilities(t *testing.T) {
	m := New(nil, 5)
	m.all = []client.ModelDetails{
		{ID: "glm-4.6", Capabilities: []string{"text", "thinking"}}, // text only
		{ID: "glm-4.6v", Capabilities: []string{"text", "vision"}},  // vision
		{ID: "totally-unknown", Capabilities: nil},                  // uncataloged
	}
	m.filter = filterText
	m.applyFilter()
	if got := len(m.table.Rows()); got != 1 {
		t.Errorf("Text filter: expected 1 row (glm-4.6), got %d", got)
	}
	m.filter = filterVision
	m.applyFilter()
	if got := len(m.table.Rows()); got != 1 {
		t.Errorf("Vision filter: expected 1 row (glm-4.6v), got %d", got)
	}
	if len(m.table.Rows()) > 0 && m.table.Rows()[0][0] != "glm-4.6v" {
		t.Errorf("Vision filter should keep glm-4.6v, got %v", m.table.Rows())
	}
}

// The Free filter no longer shows every model when Pricing is nil — that was
// the polarity bug. Only explicitly-zero-priced models match now.
func TestFreeFilterPolarity(t *testing.T) {
	m := New(nil, 5)
	m.all = []client.ModelDetails{
		{ID: "nil-pricing"}, // unknown → NOT free
		{ID: "paid", Pricing: &client.Pricing{Input: 0.1, Output: 0.2}},
		{ID: "free", Pricing: &client.Pricing{Input: 0, Output: 0}},
	}
	m.filter = filterFree
	m.applyFilter()
	if got := len(m.table.Rows()); got != 1 {
		t.Fatalf("Free filter: expected 1 row (free), got %d", got)
	}
	if m.table.Rows()[0][0] != "free" {
		t.Errorf("Free filter should keep only 'free', got %v", m.table.Rows())
	}
}

// The View must not panic for an unknown / uncataloged model (preview pane
// and table both render with sparse data).
func TestViewUncatalogedIsSafe(t *testing.T) {
	m := New(nil, 5)
	m.width, m.height = 120, 30
	m.resize()
	m.all = []client.ModelDetails{{ID: "mystery-model", OwnedBy: "z-ai"}}
	m.applyFilter()
	// Force a selection so the preview pane tries to render the unknown model.
	m.table.SetCursor(0)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("View panicked on uncataloged model: %v", r)
		}
	}()
	_ = m.View().Content
}

// Narrow terminals hide the preview pane and show the "press enter" hint
// instead — so the feature degrades gracefully on small windows.
func TestNarrowLayoutShowsEnterHint(t *testing.T) {
	m := New(nil, 5)
	m.width, m.height = 60, 20 // below twoColumnMinWidth
	m.resize()
	m.all = []client.ModelDetails{{ID: "glm-4.6"}}
	m.applyFilter()

	v := m.View()
	if !containsStr(v.Content, "enter") {
		t.Errorf("narrow layout should hint at enter-for-detail, got:\n%s", v.Content)
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
