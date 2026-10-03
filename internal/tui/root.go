package tui

import (
	"fmt"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/accounts"
	"github.com/SamyRai/go-z-ai/internal/tui/chat"
	"github.com/SamyRai/go-z-ai/internal/tui/coding"
	"github.com/SamyRai/go-z-ai/internal/tui/media"
	"github.com/SamyRai/go-z-ai/internal/tui/modelpicker"
	"github.com/SamyRai/go-z-ai/internal/tui/models"
	"github.com/SamyRai/go-z-ai/internal/tui/palette"
	"github.com/SamyRai/go-z-ai/internal/tui/tools"
	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
	"github.com/SamyRai/go-z-ai/internal/tui/usage"
)

// Layout of the chrome around the active screen: header, spacer, tab bar,
// spacer, then the bordered panel, a status line, and the help footer. The
// panel adds a border plus 1 column of padding on each side.
const (
	chromeRows     = 6
	panelVOverhead = 2
	panelHOverhead = 4
	// tabBarRow is the 0-based screen row of the tab bar (for mouse clicks).
	tabBarRow = 2
	// Below minWidth×minHeight the chrome can't render readably, so View
	// shows a resize hint instead.
	minWidth  = 60
	minHeight = 22
)

// rootModel owns the chrome, routes messages, and delegates the body to the
// active screen. Screens are constructed once, up front.
type rootModel struct {
	session     *session
	active      tab
	screens     [tabCount]tea.Model
	initialized [tabCount]bool

	width, height int

	keys keyMap
	help help.Model

	toastText  string
	toastLevel toastLevel
	toastID    int // monotonic; an expiry tick only clears a matching id

	// overlay, when non-nil, is a modal rendered over the active screen. It
	// receives keypresses first; layout, theme, routed, and status messages
	// still reach the root and screens.
	overlay tea.Model
}

func newRootModel(s *session) *rootModel {
	m := &rootModel{session: s, keys: defaultKeyMap(), help: help.New()}
	m.screens[tabChat] = chat.New(s.Client, int(tabChat))
	m.screens[tabModels] = models.New(s.Client, int(tabModels))
	m.screens[tabUsage] = usage.New(s.Client, int(tabUsage))
	m.screens[tabAccounts] = accounts.New(s.accounts, int(tabAccounts))
	m.screens[tabCoding] = coding.New(s.coding)
	m.screens[tabMedia] = media.New(s.Client, int(tabMedia))
	m.screens[tabTools] = tools.New(s.Client, int(tabTools))
	return m
}

func (m *rootModel) Init() tea.Cmd {
	m.initialized[m.active] = true
	return tea.Batch(m.screens[m.active].Init(), tea.RequestBackgroundColor)
}

// innerSize is the content area available to the active screen.
func (m *rootModel) innerSize() (int, int) {
	return max(m.width-panelHOverhead, 10), max(m.height-chromeRows-panelVOverhead, 3)
}

func (m *rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case uimsg.CloseOverlay:
		m.overlay = nil
		return m, nil

	case palette.Result:
		m.overlay = nil
		return m, m.runPaletteAction(msg)

	case uimsg.OpenModelPicker:
		m.overlay = m.openModelPicker()
		return m, m.overlay.Init()

	case modelpicker.Picked:
		m.overlay = nil
		if s, ok := m.screens[tabChat].(chatModelSetter); ok {
			ns, cmd := s.SetModel(msg.Model)
			m.screens[tabChat] = ns
			return m, tea.Batch(cmd, m.setToast("chat model: "+msg.Model, toastInfo))
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.SetWidth(msg.Width)
		innerW, innerH := m.innerSize()
		return m, m.broadcast(tea.WindowSizeMsg{Width: innerW, Height: innerH})

	case tea.BackgroundColorMsg:
		// Re-resolve the palette for the terminal's background and let
		// screens rebuild theme-dependent state.
		uistyle.SetDark(msg.IsDark())
		return m, m.broadcast(msg)

	case uimsg.Err:
		return m, m.setToast(describeErr(msg.Err))

	case uimsg.Status:
		return m, m.setToast(msg.Text, toastInfo)

	case toastExpiredMsg:
		if msg.id == m.toastID {
			m.toastText = ""
		}
		return m, nil

	case uimsg.AccountChanged:
		return m, m.switchAccount()

	case uimsg.PlanChanged:
		m.session.plan = msg.Plan
		return m, nil

	case uimsg.Routed:
		// Deliver an async result to the screen that started it, whichever
		// tab is active now.
		if msg.Tab < 0 || msg.Tab >= int(tabCount) || m.screens[msg.Tab] == nil {
			return m, nil
		}
		ns, cmd := m.screens[msg.Tab].Update(msg.Msg)
		m.screens[msg.Tab] = ns
		return m, cmd

	case tea.KeyPressMsg:
		m.toastText = "" // any key dismisses a lingering toast
		if handled, cmd := m.handleGlobalKey(msg); handled {
			return m, cmd
		}
		if m.overlay != nil {
			ns, cmd := m.overlay.Update(msg)
			m.overlay = ns
			return m, cmd
		}

	case tea.MouseClickMsg:
		m.toastText = ""
		if m.overlay != nil {
			return m, nil
		}
		if msg.Y == tabBarRow {
			if t, ok := tabBarHit(msg.X, m.active, m.width); ok {
				m.switchTab(t)
				return m, m.ensureInit()
			}
		}
	}

	if m.overlay != nil {
		ns, cmd := m.overlay.Update(msg)
		m.overlay = ns
		return m, cmd
	}
	ns, cmd := m.screens[m.active].Update(msg)
	m.screens[m.active] = ns
	return m, cmd
}

// handleGlobalKey handles the root's own shortcuts, which work with or
// without an overlay open. ctrl+c goes to a screen with an operation in
// flight (to cancel it) instead of quitting, and "?" is left to a focused
// text field. Async results are routed to their screen, so tabs and
// overlays stay usable while anything runs.
func (m *rootModel) handleGlobalKey(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		if m.activeStreaming() {
			return false, nil
		}
		return true, tea.Quit
	case key.Matches(msg, m.keys.Palette):
		if _, open := m.overlay.(palette.Model); open {
			m.overlay = nil
		} else {
			m.overlay = m.openPalette()
		}
		return true, nil
	case key.Matches(msg, m.keys.Help) && (msg.String() != "?" || m.isHelpOpen() || (m.overlay == nil && !m.activeCapturesInput())):
		if m.isHelpOpen() {
			m.overlay = nil
		} else {
			m.overlay = m.openHelpOverlay()
		}
		return true, nil
	case m.overlay != nil:
		return false, nil
	case key.Matches(msg, m.keys.NextTab):
		m.switchTab((m.active + 1) % tabCount)
		return true, m.ensureInit()
	case key.Matches(msg, m.keys.PrevTab):
		m.switchTab((m.active + tabCount - 1) % tabCount)
		return true, m.ensureInit()
	}
	return false, nil
}

func (m *rootModel) isHelpOpen() bool {
	_, ok := m.overlay.(*helpOverlay)
	return ok
}

// broadcast delivers msg to every screen and the overlay.
func (m *rootModel) broadcast(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for i, s := range m.screens {
		if s == nil {
			continue
		}
		ns, cmd := s.Update(msg)
		m.screens[i] = ns
		cmds = append(cmds, cmd)
	}
	if m.overlay != nil {
		ns, cmd := m.overlay.Update(msg)
		m.overlay = ns
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

// switchAccount points the client at the newly active account and reloads
// every screen that has already fetched data with the old one.
func (m *rootModel) switchAccount() tea.Cmd {
	acct, err := m.session.useActiveAccount()
	if err != nil {
		return m.setToast(describeErr(err))
	}
	cmds := []tea.Cmd{}
	for i := range m.screens {
		if m.initialized[i] {
			cmds = append(cmds, m.refreshScreen(tab(i)))
		}
	}
	if acct.Name != "" {
		cmds = append(cmds, m.setToast("now using account "+acct.Name, toastInfo))
	}
	return tea.Batch(cmds...)
}

// refreshScreen reloads screen t when it supports refreshing.
func (m *rootModel) refreshScreen(t tab) tea.Cmd {
	r, ok := m.screens[t].(refresher)
	if !ok {
		return nil
	}
	ns, cmd := r.Refresh()
	m.screens[t] = ns
	return cmd
}

func (m *rootModel) switchTab(t tab) {
	m.active = t
	m.toastText = ""
	m.overlay = nil // an overlay belongs to the screen that opened it
}

// ensureInit runs a screen's Init the first time it becomes active, so
// startup doesn't fire every tab's API calls.
func (m *rootModel) ensureInit() tea.Cmd {
	if m.initialized[m.active] || m.screens[m.active] == nil {
		return nil
	}
	m.initialized[m.active] = true
	return m.screens[m.active].Init()
}

func (m *rootModel) View() tea.View {
	if m.width > 0 && m.height > 0 && (m.width < minWidth || m.height < minHeight) {
		msg := fmt.Sprintf("Terminal too small (%dx%d).\nResize to at least %dx%d to use the TUI.",
			m.width, m.height, minWidth, minHeight)
		v := tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg))
		v.AltScreen = true
		return v
	}

	innerW, innerH := m.innerSize()
	panel := uistyle.Panel.Width(innerW).Height(innerH).Render(m.screens[m.active].View().Content)
	if m.overlay != nil {
		panel = placeOverlay(innerW, innerH, m.overlay.View().Content)
	}

	// The status line is always reserved so the panel never shifts.
	status := uistyle.Subtle.Render("? / f1 help · ctrl+p command palette")
	if m.toastText != "" {
		status = toastStyleFor(m.toastLevel)(m.toastText)
	}

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
		m.renderHeader(innerW),
		"",
		renderTabBar(m.active, m.width),
		"",
		panel,
		status,
		m.help.ShortHelpView(m.footerBindings()),
	))
	v.AltScreen = true
	// Cell-motion mouse mode: tab-bar clicks, and wheel scrolling for the
	// screens' viewports and lists.
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m *rootModel) footerBindings() []key.Binding {
	bindings := []key.Binding{m.keys.Help, m.keys.Palette, m.keys.NextTab, m.keys.PrevTab, m.keys.Quit}
	if h, ok := m.screens[m.active].(helpProvider); ok {
		bindings = append(h.ShortHelp(), bindings...)
	}
	return bindings
}
