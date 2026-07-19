// Package coding implements the TUI's Coding tab: install/config status,
// auth, load, and unload for supported coding-agent tools (Claude Code,
// OpenCode, Crush, Factory Droid), backed by pkg/coding — the same package
// the "go-z-ai coding" commands use.
package coding

import (
	"context"
	"fmt"
	"os"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/SamyRai/go-z-ai/internal/coding"
	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
)

type item struct {
	tool      coding.Tool
	installed bool
	detected  coding.Detection
}

func (i item) FilterValue() string { return i.tool.DisplayName }
func (i item) Title() string {
	mark := "not installed"
	if i.installed {
		mark = "installed"
		if i.detected.Configured {
			mark = "configured for Z.AI"
		}
	}
	return fmt.Sprintf("%s — %s", i.tool.DisplayName, mark)
}
func (i item) Description() string {
	if !i.detected.Configured {
		return "not configured"
	}
	return fmt.Sprintf("plan: %s", coding.DisplayName(i.detected.Plan))
}

type mode int

const (
	modeList mode = iota
	modeAuth
)

type refreshedMsg struct {
	items []item
	err   error
}

type actionDoneMsg struct{ err error }

// Model is the Coding tab's screen model.
type Model struct {
	store *coding.Store
	list  list.Model
	mode  mode

	form     *huh.Form
	formPlan string
	formKey  string
}

// New builds the Coding screen. store must be non-nil.
func New(store *coding.Store) Model {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.SetShowTitle(false)

	return Model{store: store, list: l}
}

func refresh() tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return refreshedMsg{err: err}
		}
		items := make([]item, 0, len(coding.Tools))
		for _, t := range coding.Tools {
			installed := t.IsInstalled()
			var d coding.Detection
			if installed {
				d, _ = coding.Detect(home, t.ID)
			}
			items = append(items, item{tool: t, installed: installed, detected: d})
		}
		return refreshedMsg{items: items}
	}
}

func (m Model) Init() tea.Cmd { return refresh() }

func (m *Model) newAuthForm() *huh.Form {
	m.formPlan, m.formKey = "", ""
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Plan").
				Options(huh.NewOption("Global", "global"), huh.NewOption("China", "china")).
				Value(&m.formPlan),
			huh.NewInput().
				Title("Z.AI API key").
				EchoMode(huh.EchoModePassword).
				Value(&m.formKey).
				Validate(huh.ValidateNotEmpty()),
		),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, max(msg.Height-2, 3))
		return m, nil

	case refreshedMsg:
		if msg.err != nil {
			return m, func() tea.Msg { return uimsg.Err{Err: msg.err} }
		}
		out := make([]list.Item, len(msg.items))
		for i, it := range msg.items {
			out[i] = it
		}
		m.list.SetItems(out)
		return m, nil

	case actionDoneMsg:
		if msg.err != nil {
			return m, func() tea.Msg { return uimsg.Err{Err: msg.err} }
		}
		return m, refresh()

	case tea.KeyPressMsg:
		if m.mode == modeAuth {
			return m.updateAuth(msg)
		}

		switch msg.String() {
		case "a":
			m.mode = modeAuth
			m.form = m.newAuthForm()
			return m, m.form.Init()
		case "l":
			if it, ok := m.selected(); ok {
				return m, m.loadTool(it.tool.ID)
			}
		case "u":
			if it, ok := m.selected(); ok {
				return m, m.unloadTool(it.tool.ID)
			}
		case "m":
			if it, ok := m.selected(); ok {
				return m, m.mcpTool(it.tool.ID)
			}
		case "r":
			return m, refresh()
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) selected() (item, bool) {
	it, ok := m.list.SelectedItem().(item)
	return it, ok
}

func (m Model) updateAuth(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.mode = modeList
		return m, nil
	}

	updated, cmd := m.form.Update(msg)
	m.form = updated.(*huh.Form)

	if m.form.State == huh.StateCompleted {
		m.mode = modeList
		return m, m.submitAuth()
	}
	if m.form.State == huh.StateAborted {
		m.mode = modeList
		return m, nil
	}
	return m, cmd
}

func (m Model) submitAuth() tea.Cmd {
	plan := m.formPlan
	key := m.formKey
	store := m.store
	return func() tea.Msg {
		if err := coding.ValidateAPIKey(context.Background(), plan, key); err != nil {
			return actionDoneMsg{err: err}
		}
		if err := store.SetPlan(plan); err != nil {
			return actionDoneMsg{err: err}
		}
		if err := store.SetAPIKey(key); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{}
	}
}

func (m Model) loadTool(toolID string) tea.Cmd {
	store := m.store
	return func() tea.Msg {
		c, err := store.Load()
		if err != nil {
			return actionDoneMsg{err: err}
		}
		if c.Plan == "" || c.APIKey == "" {
			return actionDoneMsg{err: fmt.Errorf("no credentials stored — press 'a' to auth first")}
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{err: coding.Load(home, toolID, c.Plan, c.APIKey)}
	}
}

func (m Model) unloadTool(toolID string) tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{err: coding.Unload(home, toolID)}
	}
}

// mcpTool registers Z.AI's Vision MCP server for toolID, using the stored
// API key — unlike loadTool this doesn't need a plan, since the MCP server
// isn't plan-routed.
func (m Model) mcpTool(toolID string) tea.Cmd {
	store := m.store
	return func() tea.Msg {
		c, err := store.Load()
		if err != nil {
			return actionDoneMsg{err: err}
		}
		if c.APIKey == "" {
			return actionDoneMsg{err: fmt.Errorf("no credentials stored — press 'a' to auth first")}
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{err: coding.LoadMCP(home, toolID, c.APIKey)}
	}
}

func (m Model) View() tea.View {
	if m.mode == modeAuth {
		return tea.NewView(m.form.View())
	}
	return tea.NewView(m.list.View())
}

// ShortHelp implements the root model's helpProvider interface.
func (m Model) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "auth")),
		key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "load")),
		key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "unload")),
		key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "vision mcp")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}
