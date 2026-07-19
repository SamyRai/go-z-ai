package tui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/accounts"
	"github.com/SamyRai/go-z-ai/internal/tui/chat"
	"github.com/SamyRai/go-z-ai/internal/tui/coding"
	"github.com/SamyRai/go-z-ai/internal/tui/media"
	"github.com/SamyRai/go-z-ai/internal/tui/models"
	"github.com/SamyRai/go-z-ai/internal/tui/tools"
	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
	"github.com/SamyRai/go-z-ai/internal/tui/usage"
)

// chrome rows: header line + tab bar + status line + help bar, plus the
// bordered panel's own top/bottom border. The status line is always reserved
// (blank when no toast is active) so the inner content area never shifts when
// a toast appears or expires. Panel padding is 1 col each side, border 1 col
// each side, so 4 columns of horizontal overhead too.
const (
	chromeRows     = 4
	panelVOverhead = 2
	panelHOverhead = 4
)

// toastTTL is how long a toast stays visible before auto-dismissing. A newer
// toast supersedes an older one (each gets its own timer and monotonic id, so
// a stale tick can't clear a fresh toast).
const toastTTL = 4 * time.Second

// toastExpiredMsg clears a toast after its TTL. It carries the id of the
// toast it was scheduled for, so a stale tick from an older toast can't
// dismiss a newer one.
type toastExpiredMsg struct{ id int }

// rootModel owns the tab bar and delegates Update/View to the active
// screen's tea.Model. Screens are constructed once, up front, from cfg — no
// screen calls getClient() or touches cobra itself.
type rootModel struct {
	cfg         Config
	active      tab
	screens     [tabCount]tea.Model
	initialized [tabCount]bool

	width, height int

	keys keyMap
	help help.Model

	toastText  string
	toastLevel toastLevel
	toastID    int // monotonic; a toast's expiry tick only clears a matching id
}

func newRootModel(cfg Config) *rootModel {
	m := &rootModel{cfg: cfg, keys: defaultKeyMap(), help: help.New()}
	m.screens[tabChat] = chat.New(cfg.Client)
	m.screens[tabModels] = models.New(cfg.Client, int(tabModels))
	m.screens[tabUsage] = usage.New(cfg.Client, cfg.Accounts, int(tabUsage))
	m.screens[tabAccounts] = accounts.New(cfg.Accounts)
	m.screens[tabCoding] = coding.New(cfg.Coding)
	m.screens[tabMedia] = media.New(cfg.Client, int(tabMedia))
	m.screens[tabTools] = tools.New(cfg.Client, int(tabTools))
	return m
}

func (m *rootModel) Init() tea.Cmd {
	m.initialized[m.active] = true
	return tea.Batch(m.screens[m.active].Init(), tea.RequestBackgroundColor)
}

// streamer is implemented by screens that need to intercept ctrl+c to cancel
// an in-flight operation (e.g. the chat screen mid-stream) instead of
// quitting the whole program on the first press.
type streamer interface {
	Streaming() bool
}

// helpProvider is implemented by screens that want their own keybindings
// shown in the footer alongside the global nav bindings.
type helpProvider interface {
	ShortHelp() []key.Binding
}

// innerSize returns the content area available to the active screen, after
// subtracting the header/tab-bar/footer rows and the bordered panel's own
// border+padding.
func (m *rootModel) innerSize() (int, int) {
	w := m.width - panelHOverhead
	h := m.height - chromeRows - panelVOverhead
	if w < 10 {
		w = 10
	}
	if h < 3 {
		h = 3
	}
	return w, h
}

func (m *rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		innerW, innerH := m.innerSize()
		inner := tea.WindowSizeMsg{Width: innerW, Height: innerH}
		var cmds []tea.Cmd
		for i, s := range m.screens {
			if s == nil {
				continue
			}
			ns, cmd := s.Update(inner)
			m.screens[i] = ns
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.BackgroundColorMsg:
		// Re-resolve the whole palette against the terminal's actual
		// background (light/dark) and rebuild every shared style, so the
		// next frame uses the new theme everywhere at once. Forward to every
		// screen too, so screens that cache theme-dependent state (e.g.
		// chat's glamour renderer) can rebuild.
		uistyle.SetDark(msg.IsDark())
		var cmds []tea.Cmd
		for i, s := range m.screens {
			if s == nil {
				continue
			}
			ns, cmd := s.Update(msg)
			m.screens[i] = ns
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.KeyPressMsg:
		activeStreaming := false
		if sc, ok := m.screens[m.active].(streamer); ok {
			activeStreaming = sc.Streaming()
		}

		// Any keypress dismisses a lingering toast (in addition to the TTL),
		// so an error/status line never lingers past the user's next action.
		m.toastText = ""

		// Tab navigation is blocked mid-stream for the same reason quit is:
		// stream messages are only delivered to the active screen, so
		// leaving the chat tab would stall the chunk pump.
		switch {
		case key.Matches(msg, m.keys.Quit) && !activeStreaming:
			return m, tea.Quit
		case key.Matches(msg, m.keys.NextTab) && !activeStreaming:
			m.switchTab((m.active + 1) % tabCount)
			return m, m.ensureInit()
		case key.Matches(msg, m.keys.PrevTab) && !activeStreaming:
			m.switchTab((m.active + tabCount - 1) % tabCount)
			return m, m.ensureInit()
		}

	case uimsg.Err:
		return m, m.setToast(describeErr(msg.Err))

	case uimsg.Status:
		return m, m.setToast(msg.Text, toastInfo)

	case toastExpiredMsg:
		// Only clear if this tick was for the currently-visible toast — a
		// newer toast (higher id) must not be dismissed by a stale tick.
		if msg.id == m.toastID {
			m.toastText = ""
		}
		return m, nil

	case uimsg.Routed:
		// An async result addressed to a specific screen — deliver it there
		// even if that screen isn't active, so switching tabs mid-operation
		// doesn't drop the result (see uimsg.Routed).
		if msg.Tab < 0 || msg.Tab >= len(m.screens) || m.screens[msg.Tab] == nil {
			return m, nil
		}
		ns, cmd := m.screens[msg.Tab].Update(msg.Msg)
		m.screens[msg.Tab] = ns
		return m, cmd
	}

	ns, cmd := m.screens[m.active].Update(msg)
	m.screens[m.active] = ns
	return m, cmd
}

func (m *rootModel) switchTab(t tab) {
	m.active = t
	m.toastText = ""
}

// setToast records a toast and schedules its auto-dismiss. Each toast gets a
// fresh monotonic id and its own tea.Tick, so a newer toast supersedes an
// older one and a stale tick can't clear the wrong toast.
func (m *rootModel) setToast(text string, level toastLevel) tea.Cmd {
	m.toastText = text
	m.toastLevel = level
	m.toastID++
	id := m.toastID
	return tea.Tick(toastTTL, func(time.Time) tea.Msg { return toastExpiredMsg{id: id} })
}

// ensureInit lazily calls Init on a tab the first time it becomes active, so
// switching tabs doesn't fire every screen's API calls on startup.
func (m *rootModel) ensureInit() tea.Cmd {
	if m.initialized[m.active] {
		return nil
	}
	m.initialized[m.active] = true
	return m.screens[m.active].Init()
}

func (m *rootModel) View() tea.View {
	innerW, innerH := m.innerSize()
	body := m.screens[m.active].View()
	panel := uistyle.Panel.Width(innerW).Height(innerH).Render(body.Content)

	header := uistyle.Header.Render("go-z-ai") + " " + uistyle.StatusBar.Render(m.accountLabel())

	help := m.help.ShortHelpView(m.footerBindings())
	// The status line is always present: the toast when one is active,
	// otherwise a subtle hint surfacing the two power-user shortcuts that
	// aren't in the per-screen help (the help overlay and the command
	// palette). Keeping the line reserved means the panel never shifts when a
	// toast appears or expires.
	var status string
	if m.toastText != "" {
		status = toastStyleFor(m.toastLevel)(m.toastText)
	} else {
		status = uistyle.Subtle.Render("? help · ctrl+p command palette")
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		renderTabBar(m.active),
		panel,
		status,
		help,
	)

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m *rootModel) accountLabel() string {
	if m.cfg.Accounts == nil {
		return ""
	}
	acct, ok := m.cfg.Accounts.ActiveAccount()
	if !ok {
		return "no active account"
	}
	return fmt.Sprintf("account: %s (%s)", acct.Name, acct.Type)
}

func (m *rootModel) footerBindings() []key.Binding {
	bindings := []key.Binding{m.keys.NextTab, m.keys.PrevTab, m.keys.Quit}
	if h, ok := m.screens[m.active].(helpProvider); ok {
		bindings = append(h.ShortHelp(), bindings...)
	}
	return bindings
}
