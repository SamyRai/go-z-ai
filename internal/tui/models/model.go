// Package models implements the TUI's Models tab: a browsable table of
// available Z.AI models, backed by the same ModelsService the "zai-client
// models" commands already use.
package models

import (
	"context"
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

type filter int

const (
	filterAll filter = iota
	filterText
	filterVision
	filterFree
)

var filterNames = [...]string{"All", "Text", "Vision", "Free"}

type fetchedMsg struct {
	models []client.ModelDetails
	err    error
}

// Model is the Models tab's screen model.
type Model struct {
	client  *client.Client
	selfTab int // this screen's tab index, used to route the fetch result back
	table   table.Model
	filter  filter
	all     []client.ModelDetails
	width   int
	height  int
	loading bool
}

// New builds the Models screen. c must be non-nil. selfTab is this screen's tab
// index in the root model, so a fetch result routes back here even if the user
// has switched away while it was loading.
func New(c *client.Client, selfTab int) Model {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "ID", Width: 28},
			{Title: "Context", Width: 10},
			{Title: "Owned By", Width: 14},
			{Title: "Input $/1K", Width: 12},
			{Title: "Output $/1K", Width: 12},
		}),
		table.WithFocused(true),
	)
	return Model{client: c, selfTab: selfTab, table: t}
}

func (m Model) Init() tea.Cmd {
	return m.route(m.fetch())
}

func (m Model) fetch() tea.Cmd {
	return func() tea.Msg {
		info, err := m.client.Models().List(context.Background())
		if err != nil {
			return fetchedMsg{err: err}
		}
		return fetchedMsg{models: info.Models}
	}
}

// route wraps cmd so its result is delivered back to this tab even if the user
// switched away mid-load (otherwise the fetch result is lost and the tab stays
// stuck "loading"). Same mechanism as the media tab.
func (m Model) route(cmd tea.Cmd) tea.Cmd {
	self := m.selfTab
	return func() tea.Msg { return uimsg.Routed{Tab: self, Msg: cmd()} }
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(max(msg.Height-4, 3))
		return m, nil

	case fetchedMsg:
		m.loading = false
		if msg.err != nil {
			return m, func() tea.Msg { return uimsg.Err{Err: msg.err} }
		}
		m.all = msg.models
		m.applyFilter()
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "r":
			m.loading = true
			return m, m.route(m.fetch())
		case "1":
			m.filter = filterAll
			m.applyFilter()
			return m, nil
		case "2":
			m.filter = filterText
			m.applyFilter()
			return m, nil
		case "3":
			m.filter = filterVision
			m.applyFilter()
			return m, nil
		case "4":
			m.filter = filterFree
			m.applyFilter()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *Model) applyFilter() {
	rows := make([]table.Row, 0, len(m.all))
	for _, md := range m.all {
		if !m.matches(md) {
			continue
		}
		in, out := "-", "-"
		if md.Pricing != nil {
			in = fmt.Sprintf("%.4f", md.Pricing.Input)
			out = fmt.Sprintf("%.4f", md.Pricing.Output)
		}
		rows = append(rows, table.Row{md.ID, fmt.Sprintf("%d", md.ContextSize), md.OwnedBy, in, out})
	}
	m.table.SetRows(rows)
}

func (m Model) matches(md client.ModelDetails) bool {
	switch m.filter {
	case filterText:
		return isTextModel(md.ID)
	case filterVision:
		return isVisionModel(md.ID)
	case filterFree:
		return md.Pricing == nil || (md.Pricing.Input == 0 && md.Pricing.Output == 0)
	default:
		return true
	}
}

// isTextModel/isVisionModel mirror pkg/client's own unexported heuristics
// (glm-*/text vs *-v/vision id patterns) closely enough for tab filtering;
// the authoritative fetch always goes through ModelsService.List.
func isTextModel(id string) bool { return !isVisionModel(id) }
func isVisionModel(id string) bool {
	for _, s := range []string{"vision", "-v", "glm-4v", "cogview", "cogvideo"} {
		if len(id) >= len(s) && contains(id, s) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (m Model) View() tea.View {
	pills := uistyle.RenderPills(int(m.filter), filterNames[:])
	body := pills + "\n" + m.table.View()
	if m.loading {
		body += "\nloading..."
	}
	return tea.NewView(body)
}

// ShortHelp implements the root model's helpProvider interface.
func (m Model) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("1", "2", "3", "4"), key.WithHelp("1-4", "filter")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}
