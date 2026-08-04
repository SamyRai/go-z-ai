package modelpicker

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

// feedModels sets the picker's catalog (skipping the async fetch) and returns
// the updated concrete Model. This is the seam Init's fetchedMsg would use.
func feedModels(m Model, ids ...string) Model {
	mds := make([]client.ModelDetails, len(ids))
	for i, id := range ids {
		mds[i] = client.ModelDetails{ID: id}
	}
	tm, _ := m.Update(fetchedMsg{models: mds})
	return tm.(Model)
}

// nextModel sends a key to m and returns the concrete resulting Model.
func nextModel(t *testing.T, m Model, key tea.KeyPressMsg) Model {
	t.Helper()
	tm, _ := m.Update(key)
	got, ok := tm.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", tm)
	}
	return got
}

func TestEmptyCatalogEnterCloses(t *testing.T) {
	m := feedModels(New(nil, "")) // no models
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a cmd on enter with empty list")
	}
	if _, ok := cmd().(uimsg.CloseOverlay); !ok {
		t.Error("enter on empty matched list should close the overlay")
	}
}

func TestEnterPicksHighlightedModel(t *testing.T) {
	m := feedModels(New(nil, ""), "glm-4.6", "glm-4.5-air")
	// Models are sorted by ID on fetch, so index 0 is glm-4.5-air (it sorts
	// before glm-4.6). current="" means no model is highlighted, cursor stays 0.
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a cmd on enter")
	}
	picked, ok := cmd().(Picked)
	if !ok {
		t.Fatalf("expected Picked msg, got %T", cmd())
	}
	if picked.Model != "glm-4.5-air" {
		t.Errorf("expected glm-4.5-air at sorted index 0, got %q", picked.Model)
	}
}

func TestDownUpCursorBounds(t *testing.T) {
	m := feedModels(New(nil, ""), "a", "b", "c")
	// Move down past the end — cursor must clamp at last index (2).
	for i := 0; i < 5; i++ {
		m = nextModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if m.cursor != 2 {
		t.Errorf("cursor should clamp at 2 (last), got %d", m.cursor)
	}
	// Move up past the start — cursor must clamp at 0.
	for i := 0; i < 5; i++ {
		m = nextModel(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	}
	if m.cursor != 0 {
		t.Errorf("cursor should clamp at 0 (first), got %d", m.cursor)
	}
}

func TestFilterNarrowsAndClampsCursor(t *testing.T) {
	m := feedModels(New(nil, ""), "glm-4.6", "glm-4.5-air", "glm-4-plus")
	m.cursor = 2
	// Typing 'z' (no model contains it) → matched empties, cursor clamps to 0.
	next := nextModel(t, m, tea.KeyPressMsg{Code: 'z', Text: "z"})
	if len(next.matched) != 0 {
		t.Errorf("expected 0 matches for 'z', got %d", len(next.matched))
	}
	if next.cursor != 0 {
		t.Errorf("cursor should clamp to 0 on empty match, got %d", next.cursor)
	}
}

func TestEmptyQueryParksCursorOnCurrent(t *testing.T) {
	m := feedModels(New(nil, "glm-4.5-air"), "glm-4.6", "glm-4.5-air", "glm-4-plus")
	// applyFilter("") runs on fetch; cursor should park on the current model.
	if m.cursor != 1 {
		t.Errorf("cursor should park on current model index 1, got %d", m.cursor)
	}
	// Enter picks the current model (glm-4.5-air).
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	picked, ok := cmd().(Picked)
	if !ok || picked.Model != "glm-4.5-air" {
		t.Errorf("expected to pick current glm-4.5-air, got %v", picked)
	}
}

func TestEscCloses(t *testing.T) {
	m := feedModels(New(nil, ""), "a")
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if _, ok := cmd().(uimsg.CloseOverlay); !ok {
		t.Error("esc should close the overlay")
	}
}

func TestFetchedErrorSetsLoadError(t *testing.T) {
	m := New(nil, "")
	tm, _ := m.Update(fetchedMsg{err: errBoom{}})
	next := tm.(Model)
	if next.loadError == "" {
		t.Error("expected loadError to be set on fetch failure")
	}
}

type errBoom struct{}

func (errBoom) Error() string { return "boom" }
