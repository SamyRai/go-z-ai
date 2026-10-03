// Package coding implements the TUI's Coding tab: install/config status,
// auth, load, unload, and MCP setup for the supported coding tools, backed by
// internal/coding — the same package the "go-z-ai coding" commands use.
package coding

import (
	"context"
	"errors"
	"fmt"
	"os"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/SamyRai/go-z-ai/internal/coding"
	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
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

// actionDoneMsg reports a finished action: an error, or an optional status
// line and (after auth) the newly stored plan.
type actionDoneMsg struct {
	err    error
	status string
	plan   string
}

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
	// Enable the list's built-in fuzzy filter (sahilm/fuzzy, already an
	// indirect dep): press '/' to start typing a query, 'esc' to clear.
	l.SetFilteringEnabled(true)

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
				d, _ = t.Detect(home)
			}
			items = append(items, item{tool: t, installed: installed, detected: d})
		}
		return refreshedMsg{items: items}
	}
}

func (m Model) Init() tea.Cmd { return refresh() }

// Refresh re-scans installed coding-agent tools. Implements the root model's
// refresher interface so the command palette's "Refresh current tab" action
// can trigger the same path as the 'r' key.
func (m Model) Refresh() (tea.Model, tea.Cmd) {
	return m, refresh()
}

func (m *Model) newAuthForm() *huh.Form {
	m.formPlan, m.formKey = "", ""
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Plan").
				Options(planOptions()...).
				Value(&m.formPlan),
			huh.NewInput().
				Title("Z.AI API key").
				EchoMode(huh.EchoModePassword).
				Value(&m.formKey).
				Validate(huh.ValidateNotEmpty()),
		),
	)
}

// planOptions lists the plans with their real identifiers — the values the
// credential store and every tool config expect.
func planOptions() []huh.Option[string] {
	opts := make([]huh.Option[string], len(coding.Plans))
	for i, p := range coding.Plans {
		opts[i] = huh.NewOption(coding.RegionLabel(p), p)
	}
	return opts
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
		cmds := []tea.Cmd{refresh()}
		if msg.status != "" {
			cmds = append(cmds, func() tea.Msg { return uimsg.Status{Text: msg.status} })
		}
		if msg.plan != "" {
			plan := msg.plan
			cmds = append(cmds, func() tea.Msg { return uimsg.PlanChanged{Plan: plan} })
		}
		return m, tea.Batch(cmds...)

	case tea.KeyPressMsg:
		if m.mode == modeAuth {
			return m.updateAuth(msg)
		}
		// While the filter is being typed into, every key belongs to it.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "a":
			m.mode = modeAuth
			m.form = m.newAuthForm()
			return m, m.form.Init()
		case "l":
			if it, ok := m.selected(); ok {
				return m, m.loadTool(it.tool)
			}
		case "u":
			if it, ok := m.selected(); ok {
				return m, m.unloadTool(it.tool)
			}
		case "m":
			if it, ok := m.selected(); ok {
				return m, m.mcpTool(it.tool)
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
		return actionDoneMsg{status: "saved " + coding.DisplayName(plan) + " credentials", plan: plan}
	}
}

// withCredentials runs fn with the home directory and stored credentials,
// failing early when none are stored.
func (m Model) withCredentials(fn func(home string, creds *coding.StoredConfig) actionDoneMsg) tea.Cmd {
	store := m.store
	return func() tea.Msg {
		creds, err := store.Load()
		if err != nil {
			return actionDoneMsg{err: err}
		}
		if !coding.IsValidPlan(creds.Plan) || creds.APIKey == "" {
			return actionDoneMsg{err: errors.New("no valid credentials stored — press 'a' to auth first")}
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return actionDoneMsg{err: err}
		}
		return fn(home, creds)
	}
}

// loadTool writes the stored plan into the tool with the same defaults the
// CLI uses.
func (m Model) loadTool(tool coding.Tool) tea.Cmd {
	return m.withCredentials(func(home string, creds *coding.StoredConfig) actionDoneMsg {
		if err := tool.Load(home, coding.NewLoadConfig(creds.Plan, creds.APIKey)); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{status: "loaded " + coding.DisplayName(creds.Plan) + " into " + tool.DisplayName}
	})
}

func (m Model) unloadTool(tool coding.Tool) tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return actionDoneMsg{err: err}
		}
		switch err := tool.Unload(home); {
		case errors.Is(err, coding.ErrNotConfigured):
			return actionDoneMsg{status: tool.DisplayName + " is not configured for a Z.AI plan"}
		case err != nil:
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{status: "removed the Z.AI plan from " + tool.DisplayName}
	}
}

// mcpTool registers every official Z.AI MCP server in the tool.
func (m Model) mcpTool(tool coding.Tool) tea.Cmd {
	return m.withCredentials(func(home string, creds *coding.StoredConfig) actionDoneMsg {
		if err := tool.LoadMCP(home, creds.Plan, creds.APIKey, coding.MCPServers); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{status: fmt.Sprintf("registered %d Z.AI MCP servers in %s", len(coding.MCPServers), tool.DisplayName)}
	})
}

func (m Model) View() tea.View {
	if m.mode == modeAuth {
		return tea.NewView(m.form.View())
	}
	// Friendly empty state: the coding tools list comes from a local scan,
	// and an empty result usually means the scan hasn't run or found nothing
	// installed — a CTA to refresh (or, after refresh, that nothing's
	// installed) reads far better than the list's bare "No items."
	if len(m.list.Items()) == 0 {
		body := uistyle.EmptyTitle.Render("No coding tools detected") + "\n\n" +
			uistyle.EmptyHint.Render("press 'r' to rescan installed agents")
		return tea.NewView(body)
	}
	return tea.NewView(m.list.View())
}

// CapturesInput reports whether a text field has focus (the auth form or the
// list filter), so the root leaves printable keys to it.
func (m Model) CapturesInput() bool {
	return m.mode == modeAuth || m.list.FilterState() == list.Filtering
}

// ShortHelp implements the root model's helpProvider interface.
func (m Model) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "auth")),
		key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "load")),
		key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "unload")),
		key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "add mcp servers")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	}
}
