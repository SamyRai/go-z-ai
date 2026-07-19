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
	"github.com/SamyRai/go-z-ai/internal/tui/modelpicker"
	"github.com/SamyRai/go-z-ai/internal/tui/models"
	"github.com/SamyRai/go-z-ai/internal/tui/palette"
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
	// tabBarRow is the screen row index (0-based) of the tab-bar strip. The
	// header occupies row 0; the tab bar is right below it. Used by the
	// mouse-click handler to hit-test tab pills.
	tabBarRow = 1
	// minWidth/minHeight are the smallest terminal at which the full chrome
	// (header + tabs + panel + status + help) renders readably. Below this,
	// View short-circuits to a centered "please resize" message instead of a
	// cramped, broken layout. The screens themselves still receive resize
	// msgs and floor their own dimensions, so we never crash.
	minWidth  = 60
	minHeight = 20
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

	// overlay, when non-nil, is a modal tea.Model rendered centered on top of
	// the active screen (help, command palette, model picker). While open it
	// receives keypresses first; resize/background-color msgs still reach the
	// screens underneath so they stay correctly laid out when the overlay
	// closes. Opening a new overlay replaces any existing one.
	overlay tea.Model
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

// refresher is implemented by screens that have a 'r' refresh binding. The
// command palette's "Refresh current tab" action calls it directly rather
// than synthesizing a keypress (which is awkward in v2). It returns the
// updated model alongside the cmd so value-receiver screens can flip their
// loading flag and have it persist.
type refresher interface {
	Refresh() (tea.Model, tea.Cmd)
}

// chatModelSetter is implemented by the chat screen: the model-picker overlay
// returns a chosen id, and the root forwards it via SetModel so the chat
// screen stays the source of truth for the active model.
type chatModelSetter interface {
	SetModel(id string) (tea.Model, tea.Cmd)
}

// chatModelGetter is implemented by the chat screen so the root can pass the
// currently-selected model id into the picker (to highlight it in the list).
type chatModelGetter interface {
	ModelID() string
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
	// An open overlay owns the keyboard; only resize / background-color /
	// routed / toast msgs still flow to the screens underneath (so they stay
	// correctly laid out and an in-flight async result still lands when the
	// overlay closes). CloseOverlay dismisses the overlay regardless of msg.
	if _, ok := msg.(uimsg.CloseOverlay); ok {
		m.overlay = nil
		return m, nil
	}

	// A command-palette Result carries an action the root must perform
	// (switch tab, refresh, toggle help, open model picker, quit). Handle it
	// before overlay routing so the palette can self-close and dispatch.
	if res, ok := msg.(palette.Result); ok {
		m.overlay = nil
		return m, m.runPaletteAction(res)
	}

	// The chat screen asks the root to open the model picker (ctrl+m, or via
	// the palette). The root owns the client and the overlay slot, so it
	// builds the picker. Blocked mid-stream like the other overlays.
	if _, ok := msg.(uimsg.OpenModelPicker); ok {
		activeStreaming := false
		if sc, ok := m.screens[m.active].(streamer); ok {
			activeStreaming = sc.Streaming()
		}
		if !activeStreaming {
			m.overlay = m.openModelPicker()
			return m, m.overlay.Init()
		}
		return m, nil
	}

	// The model picker returned a chosen id — forward it to the chat screen
	// and dismiss the overlay.
	if picked, ok := msg.(modelpicker.Picked); ok {
		m.overlay = nil
		if s, ok := m.screens[tabChat].(chatModelSetter); ok {
			ns, cmd := s.SetModel(picked.Model)
			m.screens[tabChat] = ns
			return m, tea.Batch(cmd, func() tea.Msg {
				return uimsg.Status{Text: "chat model: " + picked.Model}
			})
		}
		return m, nil
	}

	if m.overlay != nil {
		// WindowSizeMsg and BackgroundColorMsg must reach the screens beneath
		// the overlay too, otherwise they'd render at the wrong size/theme
		// the instant the overlay closes.
		switch msg.(type) {
		case tea.WindowSizeMsg, tea.BackgroundColorMsg, uimsg.Routed:
			// fall through to the normal switch, then also forward to overlay
		default:
			ns, cmd := m.overlay.Update(msg)
			m.overlay = ns
			return m, cmd
		}
	}

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
		// Overlays get the inner content area too, so they resize with the
		// terminal (and clamp their card size accordingly).
		if m.overlay != nil {
			ns, cmd := m.overlay.Update(inner)
			m.overlay = ns
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
		if m.overlay != nil {
			ns, cmd := m.overlay.Update(msg)
			m.overlay = ns
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

		// Tab navigation and global overlays are blocked mid-stream for the
		// same reason quit is: stream messages are only delivered to the
		// active screen, so leaving the chat tab would stall the chunk pump.
		switch {
		case key.Matches(msg, m.keys.Quit) && !activeStreaming:
			return m, tea.Quit
		case key.Matches(msg, m.keys.NextTab) && !activeStreaming:
			m.switchTab((m.active + 1) % tabCount)
			return m, m.ensureInit()
		case key.Matches(msg, m.keys.PrevTab) && !activeStreaming:
			m.switchTab((m.active + tabCount - 1) % tabCount)
			return m, m.ensureInit()
		case key.Matches(msg, m.keys.Help) && !activeStreaming:
			// Toggle: close if already open, else build a fresh overlay from
			// the active screen's bindings.
			if m.overlay != nil {
				m.overlay = nil
				return m, nil
			}
			m.overlay = m.openHelpOverlay()
			return m, nil
		case key.Matches(msg, m.keys.Palette) && !activeStreaming:
			// Toggle: ctrl+p again closes the palette.
			if m.overlay != nil {
				m.overlay = nil
				return m, nil
			}
			m.overlay = m.openPalette()
			return m, nil
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

	case tea.MouseClickMsg:
		// A click also dismisses a lingering toast, matching the
		// any-keypress-dismisses behavior.
		m.toastText = ""

		// Click on the tab bar (the row directly below the header) switches
		// tabs. Other clicks fall through to the active screen so viewports
		// and lists can handle them (bubbles components already forward
		// mouse msgs in their Update). Tab-bar clicks are blocked mid-stream
		// for the same reason keyboard tab-nav is.
		activeStreaming := false
		if sc, ok := m.screens[m.active].(streamer); ok {
			activeStreaming = sc.Streaming()
		}
		if msg.Y == tabBarRow && !activeStreaming && m.overlay == nil {
			if t, ok := tabBarHit(msg.X); ok {
				m.switchTab(t)
				return m, m.ensureInit()
			}
		}
	}

	ns, cmd := m.screens[m.active].Update(msg)
	m.screens[m.active] = ns
	return m, cmd
}

func (m *rootModel) switchTab(t tab) {
	m.active = t
	m.toastText = ""
	m.overlay = nil // an overlay is bound to its opening screen; don't strand it
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
	if m.screens[m.active] == nil {
		return nil
	}
	m.initialized[m.active] = true
	return m.screens[m.active].Init()
}

func (m *rootModel) View() tea.View {
	// Min-size guard: below 60x20 the chrome would overlap itself, so render
	// only a centered resize hint. WindowSizeMsg still flows to every screen
	// (they floor their own dims), so growing back past the threshold resumes
	// a correctly-laid-out app with no extra work.
	if m.width > 0 && m.height > 0 && (m.width < minWidth || m.height < minHeight) {
		msg := fmt.Sprintf("Terminal too small (%dx%d).\nResize to at least %dx%d to use the TUI.",
			m.width, m.height, minWidth, minHeight)
		centered := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
		v := tea.NewView(centered)
		v.AltScreen = true
		return v
	}

	innerW, innerH := m.innerSize()
	body := m.screens[m.active].View()
	panel := uistyle.Panel.Width(innerW).Height(innerH).Render(body.Content)

	// Composite any open overlay centered over the panel area. The overlay
	// sees the same inner content area it was sized against in Update.
	if m.overlay != nil {
		ov := m.overlay.View()
		panel = placeOverlay(innerW, innerH, ov.Content)
	}

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
		renderTabBar(m.active, m.width),
		panel,
		status,
		help,
	)

	v := tea.NewView(content)
	v.AltScreen = true
	// Enable cell-motion mouse mode (wheel, click, drag) for the whole
	// app: the tab bar uses clicks for navigation, and the screen viewports
	// / lists get wheel-scrolling for free (bubbles components already
	// forward mouse msgs to their own Update).
	v.MouseMode = tea.MouseModeCellMotion
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

// openHelpOverlay builds the help overlay from the global keymap and the
// active screen's ShortHelp bindings. Called when the user presses "?".
func (m *rootModel) openHelpOverlay() tea.Model {
	screenTitle := tabNames[m.active]
	screenBindings := []key.Binding{}
	if h, ok := m.screens[m.active].(helpProvider); ok {
		screenBindings = h.ShortHelp()
	}
	return newHelpOverlay(m.keys, screenTitle, screenBindings)
}

// openModelPicker builds the model-picker overlay from the chat screen's
// current model (to highlight it) and the root's client (to fetch the
// catalog). The picker fetches on Init; the root kicks that off after
// assigning the overlay slot.
func (m *rootModel) openModelPicker() tea.Model {
	current := ""
	if g, ok := m.screens[tabChat].(chatModelGetter); ok {
		current = g.ModelID()
	}
	return modelpicker.New(m.cfg.Client, current)
}

// openPalette builds the command-palette overlay. Tab names are passed in so
// the palette needn't import the tab enum; the static action set (refresh,
// toggle help, switch chat model, quit) is the same regardless of the active
// screen.
func (m *rootModel) openPalette() tea.Model {
	cmds := make([]palette.Command, 0, len(tabNames)+4)
	for i, name := range tabNames {
		cmds = append(cmds, palette.Command{
			Name:  "Go to " + name,
			Desc:  "switch tab",
			Hint:  "Navigation",
			Do:    palette.ActionSwitchTab,
			DoArg: i,
		})
	}
	cmds = append(cmds,
		palette.Command{Name: "Refresh current tab", Desc: "reload data", Do: palette.ActionRefresh},
		palette.Command{Name: "Toggle help", Desc: "open the keybindings overlay", Do: palette.ActionToggleHelp},
		palette.Command{Name: "Switch chat model", Desc: "open the model picker (chat tab)", Do: palette.ActionOpenModelPicker},
		palette.Command{Name: "Quit", Desc: "exit go-z-ai tui", Do: palette.ActionQuit},
	)
	return palette.New(cmds)
}

// runPaletteAction performs the action chosen from the command palette. The
// palette has already been dismissed by the caller.
func (m *rootModel) runPaletteAction(res palette.Result) tea.Cmd {
	switch res.Action {
	case palette.ActionSwitchTab:
		if res.Arg >= 0 && res.Arg < len(m.screens) && m.screens[res.Arg] != nil {
			m.switchTab(tab(res.Arg))
			return m.ensureInit()
		}
	case palette.ActionRefresh:
		// Screens that support refresh implement refresher; call it directly
		// rather than synthesizing a keypress (v2's KeyPressMsg is awkward to
		// build by hand). Screens without refresh just ignore the action.
		if r, ok := m.screens[m.active].(refresher); ok {
			ns, cmd := r.Refresh()
			m.screens[m.active] = ns
			return cmd
		}
		return nil
	case palette.ActionToggleHelp:
		m.overlay = m.openHelpOverlay()
		return nil
	case palette.ActionOpenModelPicker:
		// Switch to the chat tab (the model picker is chat-scoped) and open
		// the picker overlay. If already on chat, just open the picker.
		if m.screens[tabChat] != nil && m.active != tabChat {
			m.switchTab(tabChat)
			if cmd := m.ensureInit(); cmd != nil {
				// ensureInit returns nil once chat has been initialized; we
				// still open the picker regardless, so drop the cmd only if
				// present and chain it.
				return tea.Batch(cmd, func() tea.Msg { return uimsg.OpenModelPicker{} })
			}
		}
		return func() tea.Msg { return uimsg.OpenModelPicker{} }
	case palette.ActionQuit:
		return tea.Quit
	}
	return nil
}

func (m *rootModel) footerBindings() []key.Binding {
	bindings := []key.Binding{m.keys.Help, m.keys.Palette, m.keys.NextTab, m.keys.PrevTab, m.keys.Quit}
	if h, ok := m.screens[m.active].(helpProvider); ok {
		bindings = append(h.ShortHelp(), bindings...)
	}
	return bindings
}
