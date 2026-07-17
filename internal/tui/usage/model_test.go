package usage

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

// A successful fetch stores the data and grows one progress bar per quota limit.
func TestUsageFetchedSyncsBars(t *testing.T) {
	m := New(nil, nil, 5)
	m.loading = true

	next, _ := m.Update(fetchedMsg{quota: &client.QuotaLimitResponse{
		Success: true,
		Data:    client.QuotaData{Level: "pro", Limits: []client.QuotaLimit{{}, {}}},
	}})
	got := next.(Model)
	if got.loading {
		t.Error("expected loading cleared")
	}
	if got.quota == nil {
		t.Fatal("expected quota stored")
	}
	if len(got.bars) != 2 {
		t.Errorf("expected one bar per limit (2), got %d", len(got.bars))
	}
}

func TestUsageFetchedErrorRaisesToast(t *testing.T) {
	m := New(nil, nil, 5)
	_, cmd := m.Update(fetchedMsg{err: errors.New("boom")})
	if cmd == nil {
		t.Fatal("expected a uimsg.Err command")
	}
	if _, ok := cmd().(uimsg.Err); !ok {
		t.Error("expected uimsg.Err")
	}
}

// route addresses the fetch result to this tab so it survives a tab switch.
func TestUsageRouteWrapsToSelfTab(t *testing.T) {
	m := New(nil, nil, 3)
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

// 'r' refreshes (loading + fetch); a tick schedules another fetch+tick.
func TestUsageRefreshAndTick(t *testing.T) {
	m := New(nil, nil, 5)
	next, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if !next.(Model).loading || cmd == nil {
		t.Error("expected 'r' to set loading and return a fetch command")
	}
	if _, tickCmd := m.Update(tickMsg{}); tickCmd == nil {
		t.Error("expected tick to return a batched fetch+tick command")
	}
}
