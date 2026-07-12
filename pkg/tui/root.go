package tui

import (
	"fmt"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/SamyRai/go-z-ai/pkg/tui/accounts"
	"github.com/SamyRai/go-z-ai/pkg/tui/chat"
	"github.com/SamyRai/go-z-ai/pkg/tui/coding"
	"github.com/SamyRai/go-z-ai/pkg/tui/media"
	"github.com/SamyRai/go-z-ai/pkg/tui/models"
	"github.com/SamyRai/go-z-ai/pkg/tui/tools"
	"github.com/SamyRai/go-z-ai/pkg/tui/uimsg"
	"github.com/SamyRai/go-z-ai/pkg/tui/uistyle"
	"github.com/SamyRai/go-z-ai/pkg/tui/usage"
)

// chrome rows: header line + tab bar + footer, plus the bordered panel's own
// top/bottom border. Panel padding is 1 col each side, border 1 col each
// side, so 4 columns of horizontal overhead too.
const (
	chromeRows     = 3
	panelVOverhead = 2
	panelHOverhead = 4
)

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
}

func newRootModel(cfg Config) *rootModel {
	m := &rootModel{cfg: cfg, keys: defaultKeyMap(), help: help.New()}
	m.screens[tabChat] = chat.New(cfg.Client)
	m.screens[tabModels] = models.New(cfg.Client)
	m.screens[tabUsage] = usage.New(cfg.Client, cfg.Accounts)
	m.screens[tabAccounts] = accounts.New(cfg.Accounts)
	m.screens[tabCoding] = coding.New(cfg.Coding)
	m.screens[tabMedia] = media.New(cfg.Client)
	m.screens[tabTools] = tools.New(cfg.Client)
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

	case tea.KeyPressMsg:
		activeStreaming := false
		if sc, ok := m.screens[m.active].(streamer); ok {
			activeStreaming = sc.Streaming()
		}

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
		m.toastText, m.toastLevel = describeErr(msg.Err)
		return m, nil

	case uimsg.Status:
		m.toastText, m.toastLevel = msg.Text, toastInfo
		return m, nil
	}

	ns, cmd := m.screens[m.active].Update(msg)
	m.screens[m.active] = ns
	return m, cmd
}

func (m *rootModel) switchTab(t tab) {
	m.active = t
	m.toastText = ""
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

	header := uistyle.Header.Render("zai-client") + " " + uistyle.StatusBar.Render(m.accountLabel())

	footer := m.help.ShortHelpView(m.footerBindings())
	if m.toastText != "" {
		footer = toastStyleFor(m.toastLevel)(m.toastText)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		renderTabBar(m.active),
		panel,
		footer,
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
