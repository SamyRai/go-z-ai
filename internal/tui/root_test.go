package tui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/palette"
	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
)

// spyScreen is a minimal tea.Model that records the messages it receives, so a
// test can assert whether the root model delivered a message to it. A nil got
// pointer makes the spy a no-op (used when a test only needs a non-nil screen
// in a slot, not message capture).
type spyScreen struct {
	got *[]tea.Msg
}

func (s spyScreen) Init() tea.Cmd { return nil }
func (s spyScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s.got != nil {
		*s.got = append(*s.got, msg)
	}
	return s, nil
}
func (s spyScreen) View() tea.View { return tea.NewView("") }

// A uimsg.Routed message must be delivered to its target screen even when a
// different tab is active — this is what keeps an async result (e.g. a video
// generation that finishes after the user switched tabs) from being dropped.
func TestRoutedDeliversToInactiveScreen(t *testing.T) {
	// A minimal rootModel — newRootModel eagerly builds every real screen
	// (which needs live stores); the Routed dispatch only touches m.screens.
	m := &rootModel{active: tabChat}

	var received []tea.Msg
	m.screens[tabMedia] = spyScreen{got: &received}

	m.Update(uimsg.Routed{Tab: int(tabMedia), Msg: "ping"})

	if len(received) != 1 || received[0] != "ping" {
		t.Fatalf("expected the inactive target screen to receive \"ping\", got %v", received)
	}
}

// An out-of-range or nil-screen Routed tab is ignored rather than panicking.
func TestRoutedIgnoresBadTab(t *testing.T) {
	m := &rootModel{active: tabChat}
	// Should not panic: out of range low/high, and an in-range but nil screen.
	m.Update(uimsg.Routed{Tab: 999, Msg: "ping"})
	m.Update(uimsg.Routed{Tab: -1, Msg: "ping"})
	m.Update(uimsg.Routed{Tab: int(tabMedia), Msg: "ping"})
}

// openHelpOverlay builds a non-nil overlay from the active screen's bindings,
// and a CloseOverlay msg dismisses whatever overlay is set. Together these
// cover the help-key toggle path without depending on synthesizing a v2
// KeyPressMsg (whose Key.String() delegates to the uv layer and is awkward to
// build by hand in a unit test).
func TestHelpOverlayOpenAndClose(t *testing.T) {
	m := &rootModel{active: tabChat, keys: defaultKeyMap()}
	m.screens[tabChat] = spyScreen{}

	if m.overlay != nil {
		t.Fatalf("overlay should start nil")
	}
	m.overlay = m.openHelpOverlay()
	if m.overlay == nil {
		t.Fatalf("expected openHelpOverlay to return a non-nil model")
	}
	// A CloseOverlay msg clears the slot (the help overlay self-closes via it).
	m.Update(uimsg.CloseOverlay{})
	if m.overlay != nil {
		t.Fatalf("expected CloseOverlay to clear the overlay slot")
	}
}

// A toast's TTL expiry only clears the currently-visible toast; a stale tick
// from an older toast must not dismiss a newer one (the id guards against it).
func TestToastExpiryDoesNotClobberNewer(t *testing.T) {
	m := &rootModel{active: tabChat, keys: defaultKeyMap()}

	// First toast -> id 1.
	m.Update(uimsg.Status{Text: "first"})
	if m.toastText != "first" || m.toastID != 1 {
		t.Fatalf("expected first toast set, got text=%q id=%d", m.toastText, m.toastID)
	}
	// Second toast -> id 2 (supersedes).
	m.Update(uimsg.Status{Text: "second"})
	if m.toastText != "second" || m.toastID != 2 {
		t.Fatalf("expected second toast set, got text=%q id=%d", m.toastText, m.toastID)
	}
	// A stale tick for id 1 arrives — must NOT clear the id-2 toast.
	m.Update(toastExpiredMsg{id: 1})
	if m.toastText != "second" {
		t.Fatalf("stale tick must not clear the newer toast, got %q", m.toastText)
	}
	// The matching tick (id 2) does clear it.
	m.Update(toastExpiredMsg{id: 2})
	if m.toastText != "" {
		t.Fatalf("matching tick must clear the toast, got %q", m.toastText)
	}
}

// The command palette's SwitchTab action must switch the active tab, exactly
// like the keyboard tab-nav does.
func TestPaletteSwitchesTab(t *testing.T) {
	m := &rootModel{active: tabChat, keys: defaultKeyMap()}
	m.screens[tabModels] = spyScreen{}

	m.Update(palette.Result{Action: palette.ActionSwitchTab, Arg: int(tabModels)})

	if m.active != tabModels {
		t.Fatalf("expected active tab Models, got %v", m.active)
	}
}

// Below the min terminal size, View short-circuits to a resize hint instead of
// rendering the full chrome (which would overlap on a tiny terminal).
func TestMinSizeGuardRendersMessage(t *testing.T) {
	m := &rootModel{active: tabChat, keys: defaultKeyMap(), width: 40, height: 10}
	m.screens[tabChat] = spyScreen{}

	v := m.View()
	if !strings.Contains(v.Content, "too small") {
		t.Errorf("expected a resize hint below min size, got:\n%s", v.Content)
	}
	want := fmt.Sprintf("%dx%d", minWidth, minHeight)
	if !strings.Contains(v.Content, want) {
		t.Errorf("expected the hint to mention the minimum dimensions %s, got:\n%s", want, v.Content)
	}
}

// A mouse click on the tab-bar row switches tabs (hit-tested by x coordinate).
func TestMouseClickSwitchesTab(t *testing.T) {
	m := &rootModel{active: tabChat, keys: defaultKeyMap()}
	m.screens[tabChat] = spyScreen{}

	// The first pill (Chat) starts at x=0; the second (Models) follows at
	// x = pillWidth("Chat") = len("Chat")+4 = 8.
	modelsX := pillWidth("Chat")
	m.Update(tea.MouseClickMsg{X: modelsX, Y: tabBarRow})

	if m.active != tabModels {
		t.Fatalf("expected click on Models pill to switch to Models, got %v", m.active)
	}
}

// A click below the tab bar must NOT switch tabs — it falls through to the
// active screen (which forwards mouse msgs to viewports/lists).
func TestMouseClickBelowTabBarDoesNotSwitch(t *testing.T) {
	m := &rootModel{active: tabChat, keys: defaultKeyMap()}
	m.screens[tabChat] = spyScreen{}

	m.Update(tea.MouseClickMsg{X: 0, Y: tabBarRow + 5})

	if m.active != tabChat {
		t.Fatalf("expected tab unchanged on a non-tab-bar click, got %v", m.active)
	}
}

// sanity-check that the default keymap has the Help and Palette bindings, so a
// future refactor that drops them is caught (the '?' and ctrl+p handlers in
// Update depend on these existing).
func TestDefaultKeymapHasHelpAndPalette(t *testing.T) {
	km := defaultKeyMap()
	for _, b := range []key.Binding{km.Help, km.Palette, km.Quit, km.NextTab, km.PrevTab} {
		if len(b.Keys()) == 0 {
			t.Errorf("expected non-empty keys for binding %+v", b)
		}
	}
}

// chatModelSpy is a minimal tea.Model that also implements chatModelGetter, so
// renderHeader's model badge can read its ModelID without a real chat screen.
type chatModelSpy struct{ model string }

func (s chatModelSpy) Init() tea.Cmd                       { return nil }
func (s chatModelSpy) Update(tea.Msg) (tea.Model, tea.Cmd) { return s, nil }
func (s chatModelSpy) View() tea.View                      { return tea.NewView("") }
func (s chatModelSpy) ModelID() string                     { return s.model }

// renderHeader on a wide terminal shows the app badge plus every context
// badge (account name, type, plan when set, model). With no account it shows
// a warn-styled "none" so the missing-account state is unmissable.
func TestRenderHeaderShowsBadges(t *testing.T) {
	m := &rootModel{active: tabChat, keys: defaultKeyMap(), width: 100, height: 30}
	m.screens[tabChat] = chatModelSpy{model: "glm-5.2"}

	hdr := m.renderHeader(100)
	if !strings.Contains(hdr, "go-z-ai") {
		t.Errorf("expected app badge, got:\n%s", hdr)
	}
	if !strings.Contains(hdr, "none") {
		t.Errorf("expected 'none' account badge with no store, got:\n%s", hdr)
	}
	if !strings.Contains(hdr, "glm-5.2") {
		t.Errorf("expected the chat model in the header, got:\n%s", hdr)
	}
}

// On a narrow terminal the header drops the less-essential badges (type, plan)
// but keeps the account badge and the model badge so the essentials survive.
func TestRenderHeaderNarrowDropsBadges(t *testing.T) {
	m := &rootModel{active: tabChat, keys: defaultKeyMap(), width: 50, height: 30}
	m.screens[tabChat] = chatModelSpy{model: "glm-5.2"}

	hdr := m.renderHeader(50)
	if !strings.Contains(hdr, "go-z-ai") {
		t.Errorf("app badge must always be present, got:\n%s", hdr)
	}
	// At 50 cols (below compactTabWidth) the header keeps account + model.
	if !strings.Contains(hdr, "glm-5.2") {
		t.Errorf("model badge should survive at 50 cols, got:\n%s", hdr)
	}
}

// joinBadges skips empty strings and joins the rest with the separator.
func TestJoinBadges(t *testing.T) {
	got := joinBadges("  ", "a", "", "b", "", "")
	if got != "a  b" {
		t.Errorf("expected 'a  b', got %q", got)
	}
	if s := joinBadges("  "); s != "" {
		t.Errorf("expected empty join for all-empty input, got %q", s)
	}
}
