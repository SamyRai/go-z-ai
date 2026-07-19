// Package models implements the TUI's Models tab: a browsable, enriched view
// of available Z.AI models, backed by the same ModelsService the "go-z-ai
// models" commands already use.
//
// Layout is hybrid:
//   - A scannable table of MODEL | CONTEXT | IN/1M | OUT/1M | CAPS is always
//     shown.
//   - In wide terminals (>= twoColumnMinWidth cols), a live preview pane on
//     the right renders the highlighted row's full detail card and updates as
//     the user moves through the table. (Mirrors the Usage tab's responsive
//     two-column pattern.)
//   - On any terminal, pressing Enter opens a full-screen scrollable detail
//     view — essential on narrow terminals where the pane is hidden. Esc
//     returns to the table.
//
// All metadata (context, pricing, capabilities) comes from the catalog
// enrichment layered onto /models in pkg/client (see models_catalog.go); the
// bare /models endpoint returns only {id, object, created, owned_by}.
package models

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

// twoColumnMinWidth is the terminal width at which the table splits into
// table + live preview pane. Below this, only the table is shown and the user
// opens the full detail with Enter. Same threshold name as the Usage tab.
const (
	twoColumnMinWidth = 100
	previewWidth      = 42 // fixed pane width; the table takes the rest
	tableColsTotal    = 28 + 9 + 8 + 9 + 12 // ID(28) Context(8) IN(9) OUT(9) CAPS(12) ≈ table min
)

type filter int

const (
	filterAll filter = iota
	filterText
	filterVision
	filterFree
)

var filterNames = [...]string{"All", "Text", "Vision", "Free"}

type mode int

const (
	modeTable mode = iota
	modeDetail
)

type fetchedMsg struct {
	models []client.ModelDetails
	err    error
}

// Model is the Models tab's screen model.
type Model struct {
	client  *client.Client
	selfTab int // this screen's tab index, used to route the fetch result back
	table   table.Model
	view    viewport.Model // scrollable container for the detail view
	filter  filter
	mode    mode
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
			{Title: "MODEL", Width: 28},
			{Title: "CONTEXT", Width: 8},
			{Title: "IN/1M", Width: 8},
			{Title: "OUT/1M", Width: 8},
			{Title: "CAPS", Width: 12},
		}),
		table.WithFocused(true),
	)
	return Model{
		client:  c,
		selfTab: selfTab,
		table:   t,
		view:    viewport.New(),
	}
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
		m.resize()
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
		// Detail mode handles its own keys, then bounces everything else to
		// the viewport (so pgup/pgdn scroll the detail).
		if m.mode == modeDetail {
			return m.updateDetail(msg)
		}
		return m.updateTable(msg)
	}

	// In detail mode, route viewport-bound messages (mouse wheel, etc.) to
	// the viewport. In table mode, route to the table.
	var cmd tea.Cmd
	if m.mode == modeDetail {
		m.view, cmd = m.view.Update(msg)
	} else {
		m.table, cmd = m.table.Update(msg)
	}
	return m, cmd
}

// resize recomputes the table height/width and the detail viewport size from
// the current terminal dimensions. The table takes the full width when no
// preview pane is shown; when the pane is shown, the table is narrowed to
// width - previewWidth - separator.
func (m *Model) resize() {
	tableW := m.width
	if m.width >= twoColumnMinWidth {
		tableW = m.width - previewWidth - 2 // 2-col gutter between table and pane
	}
	if tableW < tableColsTotal {
		tableW = tableColsTotal
	}
	m.table.SetWidth(tableW)
	m.table.SetHeight(max(m.height-2, 3)) // leave a row for the filter pills

	// Detail viewport fills the whole content area.
	m.view.SetWidth(m.width)
	m.view.SetHeight(max(m.height-1, 3))
}

// updateTable handles key presses while in the table+preview layout.
func (m Model) updateTable(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
	case "enter":
		// Open the full-screen detail view for the highlighted row. Works on
		// every terminal width — essential when the preview pane is hidden.
		if md, ok := m.selectedModel(); ok {
			m.mode = modeDetail
			m.view.SetContent(renderDetail(md, m.width))
			m.view.GotoTop()
		}
		return m, nil
	}

	// Arrow/j/k movement routes to the table; the preview pane re-renders
	// from View() on every move (it reads the current selection), so there's
	// no extra message to emit.
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// updateDetail handles keys while in the full-screen detail view. Esc/Enter
// return to the table; everything else (pgup/pgdn, arrows, mouse) scrolls.
func (m Model) updateDetail(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.mode = modeTable
		return m, nil
	}
	var cmd tea.Cmd
	m.view, cmd = m.view.Update(msg)
	return m, cmd
}

// applyFilter rebuilds the table rows from m.all, keeping only the models
// matching the active filter. Pricing columns now come from the enriched
// catalog (per-1M tokens, USD) — the previous "$/1K" header was a lie that
// formatted the per-1M value with %.4f.
func (m *Model) applyFilter() {
	rows := make([]table.Row, 0, len(m.all))
	for _, md := range m.all {
		if !m.matches(md) {
			continue
		}
		in, out := "—", "—"
		if md.Pricing != nil {
			in = formatPrice(md.Pricing.Input)
			out = formatPrice(md.Pricing.Output)
		}
		rows = append(rows, table.Row{
			md.ID,
			formatContext(md.ContextSize),
			in,
			out,
			formatCaps(md.Capabilities),
		})
	}
	m.table.SetRows(rows)
}

func (m Model) matches(md client.ModelDetails) bool {
	switch m.filter {
	case filterText:
		// "Text" tab shows chat-capable models that are NOT vision models —
		// i.e. pure-text models. (Vision models also chat, but users picking
		// the Text tab want the non-vision subset.)
		return md.HasCapability(client.CapText) && !md.HasCapability(client.CapVision)
	case filterVision:
		return md.HasCapability(client.CapVision)
	case filterFree:
		// IsFree treats nil Pricing as "unknown" (not free) — fixes the
		// polarity bug where the old TUI showed every model here.
		return md.IsFree()
	default:
		return true
	}
}

// selectedModel returns the ModelDetails for the currently highlighted table
// row, looked up by ID (the table stores display strings, not structs). ok is
// false when there are no rows or the ID can't be resolved.
func (m Model) selectedModel() (client.ModelDetails, bool) {
	row := m.table.SelectedRow()
	if len(row) == 0 {
		return client.ModelDetails{}, false
	}
	id := row[0]
	for _, md := range m.all {
		if md.ID == id {
			return md, true
		}
	}
	return client.ModelDetails{}, false
}

func (m Model) View() tea.View {
	if m.mode == modeDetail {
		return tea.NewView(m.view.View())
	}

	pills := uistyle.RenderPills(int(m.filter), filterNames[:])
	var body string
	if m.width >= twoColumnMinWidth {
		// Wide: table on the left, live preview pane on the right.
		tableArea := m.table.View()
		preview := ""
		switch {
		case m.loading && len(m.all) == 0:
			preview = uistyle.EmptyTitle.Render("Loading models…") + "\n" +
				strings.Repeat(uistyle.SkeletonRow(previewWidth-2)+"\n", 3)
		case len(m.all) == 0:
			preview = uistyle.EmptyTitle.Render("No models yet") + "\n" +
				uistyle.EmptyHint.Render("press r to refresh")
		default:
			if md, ok := m.selectedModel(); ok {
				preview = renderPreview(md, previewWidth)
			} else {
				preview = uistyle.Subtle.Render("select a row")
			}
		}
		tableCol := lipgloss.NewStyle().Width(m.width - previewWidth - 2).Render(tableArea)
		previewCol := lipgloss.NewStyle().Width(previewWidth).Render(preview)
		body = pills + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, tableCol, "  ", previewCol)
	} else {
		// Narrow: table only; Enter opens the full detail.
		hint := uistyle.Subtle.Render("press enter for full detail")
		body = pills + "\n" + m.table.View() + "\n" + hint
	}

	if m.loading && len(m.all) > 0 {
		body += "\n" + uistyle.Subtle.Render("refreshing…")
	}
	return tea.NewView(body)
}

// ShortHelp implements the root model's helpProvider interface.
func (m Model) ShortHelp() []key.Binding {
	if m.mode == modeDetail {
		return []key.Binding{
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back to list")),
		}
	}
	return []key.Binding{
		key.NewBinding(key.WithKeys("1", "2", "3", "4"), key.WithHelp("1-4", "filter")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "detail")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}
