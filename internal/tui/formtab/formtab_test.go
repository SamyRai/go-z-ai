package formtab

import (
	"context"
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

func newTab(run func(context.Context, *client.Client, string) (string, error)) Model {
	return New(func() *client.Client { return nil }, 4,
		Form{Name: "A", Run: run},
		Form{Name: "B", Run: run},
	)
}

func echo(_ context.Context, _ *client.Client, v string) (string, error) { return "got " + v, nil }

func typeText(m Model, s string) Model {
	for _, r := range s {
		next, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = next.(Model)
	}
	return m
}

// Typing reaches the active form's input — the inputs used to be created
// unfocused, so every keystroke was silently dropped.
func TestTypingReachesFocusedInput(t *testing.T) {
	m := typeText(newTab(echo), "hi")
	if got := m.inputs[0].Value(); got != "hi" {
		t.Fatalf("input = %q, want hi", got)
	}
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = typeText(next.(Model), "yo")
	if m.active != 1 || m.inputs[1].Value() != "yo" || m.inputs[0].Value() != "hi" {
		t.Errorf("focus did not follow the active form: active=%d values=%q/%q", m.active, m.inputs[0].Value(), m.inputs[1].Value())
	}
}

// enter runs the form and routes the result to this tab; the result clears
// the busy state, and a second enter while busy is ignored.
func TestSubmitRoutesResult(t *testing.T) {
	m := typeText(newTab(echo), "x")
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(Model)
	if !m.busy || m.cancel == nil || cmd == nil {
		t.Fatal("enter should start work with a cancel func")
	}
	if _, again := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); again != nil {
		t.Error("enter while busy must be ignored")
	}

	batch, ok := cmd().(tea.BatchMsg)
	if !ok || len(batch) == 0 {
		t.Fatalf("expected a batch, got %T", cmd())
	}
	routed, ok := batch[0]().(uimsg.Routed)
	if !ok || routed.Tab != 4 {
		t.Fatalf("result not routed to tab 4: %#v", routed)
	}
	if res := routed.Msg.(resultMsg); res.text != "got x" {
		t.Errorf("result = %+v", res)
	}
	done, _ := m.Update(routed.Msg)
	if done.(Model).busy || done.(Model).result.GetContent() != "got x" {
		t.Error("result should clear busy and show the text")
	}
}

// esc cancels the in-flight request, and the resulting context.Canceled is
// not reported as an error.
func TestEscCancels(t *testing.T) {
	m := typeText(newTab(echo), "x")
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(Model)
	cancelled := false
	m.cancel = func() { cancelled = true }
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !cancelled {
		t.Fatal("esc should cancel")
	}
	done, _ := next.(Model).Update(resultMsg{err: context.Canceled})
	if done.(Model).result.GetContent() != "cancelled" || done.(Model).busy {
		t.Error("cancellation should end the work and keep the notice")
	}
	failed, _ := next.(Model).Update(resultMsg{err: errors.New("boom")})
	if failed.(Model).busy {
		t.Error("an error should end the work")
	}
}
