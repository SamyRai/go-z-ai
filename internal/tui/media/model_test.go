package media

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
)

// sized returns a media model whose result viewport has real dimensions, as it
// would after the root model's startup WindowSizeMsg — without a size the
// viewport renders its content as empty.
func sized(m Model) Model {
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return next.(Model)
}

// A terminal resultMsg clears busy and shows the text.
func TestResultMsgClearsBusy(t *testing.T) {
	m := sized(New(nil, 5))
	m.busy = true
	m.cancel = func() {}

	next, _ := m.Update(resultMsg{text: "all done"})
	got := next.(Model)
	if got.busy {
		t.Error("expected busy cleared after a result")
	}
	if got.cancel != nil {
		t.Error("expected cancel cleared after a result")
	}
	if !strings.Contains(got.View().Content, "all done") {
		t.Errorf("expected result text in view, got:\n%s", got.View().Content)
	}
}

// A real error surfaces as an error line.
func TestResultMsgError(t *testing.T) {
	m := sized(New(nil, 5))
	m.busy = true

	next, _ := m.Update(resultMsg{err: errors.New("boom")})
	if content := next.(Model).View().Content; !strings.Contains(content, "error: boom") {
		t.Errorf("expected error line, got:\n%s", content)
	}
}

// A cancellation is user-initiated, not a failure: it must clear busy without
// printing an error (the esc handler already showed "cancelled").
func TestResultMsgCanceledIsNotAnError(t *testing.T) {
	m := sized(New(nil, 5))
	m.busy = true
	m.result.SetContent("cancelled")

	next, _ := m.Update(resultMsg{err: context.Canceled})
	got := next.(Model)
	if got.busy {
		t.Error("expected busy cleared after cancellation")
	}
	if strings.Contains(got.View().Content, "error:") {
		t.Errorf("cancellation must not render as an error, got:\n%s", got.View().Content)
	}
}

// Enter starts an operation: busy is set and a cancel func is recorded so esc
// can abort it. (The returned Cmd is not executed here — that would hit the
// network — so a nil client is fine.)
func TestEnterStartsWorkAndRecordsCancel(t *testing.T) {
	m := sized(New(nil, 5))

	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := next.(Model)
	if !got.busy {
		t.Error("expected busy set after enter")
	}
	if got.cancel == nil {
		t.Error("expected a cancel func recorded after enter")
	}
	if cmd == nil {
		t.Error("expected a command to be returned for the async work")
	}
}

// Esc while busy invokes the stored cancel func.
func TestEscCancelsInFlightWork(t *testing.T) {
	m := sized(New(nil, 5))
	cancelled := false
	m.busy = true
	m.cancel = func() { cancelled = true }

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !cancelled {
		t.Error("expected esc to call the cancel func")
	}
	if !strings.Contains(next.(Model).View().Content, "cancelled") {
		t.Error("expected a 'cancelled' notice after esc")
	}
}

// Enter while already busy is a no-op — it must not replace the in-flight
// operation's cancel func (which would strand the running goroutine).
func TestEnterIgnoredWhileBusy(t *testing.T) {
	m := sized(New(nil, 5))
	orig := func() {}
	m.busy = true
	m.cancel = orig

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected no new command when enter is pressed while busy")
	}
}

// submit wraps its result in uimsg.Routed addressed to this screen's tab, so
// the root model can deliver it back even if the user has switched tabs. The
// OCR error path (a missing local file) reaches the wrapper without needing a
// client, so it exercises the routing envelope without hitting the network.
func TestSubmitResultIsRoutedToSelfTab(t *testing.T) {
	m := sized(New(nil, 5))
	m.active = formOCR
	m.inputs[formOCR].SetValue("/nonexistent/path/x.png")

	msg := m.submit()()
	routed, ok := msg.(uimsg.Routed)
	if !ok {
		t.Fatalf("expected a uimsg.Routed, got %T", msg)
	}
	if routed.Tab != 5 {
		t.Errorf("expected result routed to tab 5, got %d", routed.Tab)
	}
	if _, ok := routed.Msg.(resultMsg); !ok {
		t.Errorf("expected wrapped resultMsg, got %T", routed.Msg)
	}
}
