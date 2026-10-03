package usage

import (
	"errors"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

// A successful fetch stores the data and grows one progress bar per quota limit.
func TestUsageFetchedSyncsBars(t *testing.T) {
	m := New(nil, 5)
	m.loading = true

	next, _ := m.Update(fetchedMsg{quota: &client.QuotaLimitResponse{
		Data: client.QuotaData{Level: "pro", Limits: []client.QuotaLimit{{}, {}}},
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

// A failed fetch is shown in the tab (no toast every refresh), e.g. for a
// pay-as-you-go key that has no coding-plan quota.
func TestUsageFetchedErrorShownInline(t *testing.T) {
	m := New(nil, 5)
	next, cmd := m.Update(fetchedMsg{err: errors.New("boom")})
	if cmd != nil {
		t.Error("a fetch error must not raise a toast")
	}
	if next.(Model).err == nil {
		t.Error("expected the error kept for display")
	}
}

// The refresh tick is routed to this tab, so auto-refresh survives a tab
// switch.
func TestUsageTickIsRouted(t *testing.T) {
	m := New(nil, 3)
	m.every = time.Millisecond
	routed, ok := m.tick()().(uimsg.Routed)
	if !ok || routed.Tab != 3 {
		t.Fatalf("tick not routed to tab 3: %#v", routed)
	}
}

// 'r' refreshes (loading + fetch); a tick schedules another fetch+tick.
func TestUsageRefreshAndTick(t *testing.T) {
	m := New(func() *client.Client { return nil }, 5)
	next, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if !next.(Model).loading || cmd == nil {
		t.Error("expected 'r' to set loading and return a fetch command")
	}
	if _, tickCmd := m.Update(tickMsg{}); tickCmd == nil {
		t.Error("expected tick to return a batched fetch+tick command")
	}
}
