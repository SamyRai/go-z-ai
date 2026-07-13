// Package usage implements the TUI's Usage tab: a live quota + token/tool
// usage dashboard, backed by the same QuotaService/UsageService the
// "zai-client usage"/"accounts quota"/"accounts usage" commands use. It
// replaces the CLI's naive time.Sleep polling loop ("usage check --watch")
// with a non-blocking tea.Tick refresh.
package usage

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/SamyRai/go-z-ai/pkg/accounts"
	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/SamyRai/go-z-ai/pkg/tui/uimsg"
	"github.com/SamyRai/go-z-ai/pkg/tui/uistyle"
	"github.com/SamyRai/go-z-ai/pkg/usageview"
)

const (
	refreshInterval = 30 * time.Second
	// twoColumnMinWidth is the terminal width below which the quota panel
	// and the heatmap panel stack vertically instead of side by side.
	twoColumnMinWidth = 100
)

type tickMsg time.Time

type fetchedMsg struct {
	quota  *client.QuotaLimitResponse
	status *client.AccountStatus
	models *client.ModelUsageResponse
	tools  *client.ToolUsageResponse
	err    error
}

// Model is the Usage tab's screen model.
type Model struct {
	client   *client.Client
	accounts *accounts.Store

	quota  *client.QuotaLimitResponse
	status *client.AccountStatus
	models *client.ModelUsageResponse
	tools  *client.ToolUsageResponse
	bars   []progress.Model // one per m.quota.Data.Limits entry

	loading bool
	width   int
}

// New builds the Usage screen. c must be non-nil; store may be nil if no
// account store is available.
func New(c *client.Client, store *accounts.Store) Model {
	return Model{client: c, accounts: store}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.fetch(), tick())
}

func tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) fetch() tea.Cmd {
	c := m.client
	return func() tea.Msg {
		ctx := context.Background()
		var out fetchedMsg

		quota, err := c.Quota().GetQuotaLimit(ctx)
		if err != nil {
			return fetchedMsg{err: err}
		}
		out.quota = quota

		status, err := c.Usage().GetAccountStatus(ctx)
		if err == nil {
			out.status = status
		}

		start, end := usageview.Window(14, false)
		if models, err := c.Quota().GetModelUsage(ctx, start, end); err == nil {
			out.models = models
		}
		if tools, err := c.Quota().GetToolUsage(ctx, start, end); err == nil {
			out.tools = tools
		}

		return out
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tickMsg:
		return m, tea.Batch(m.fetch(), tick())

	case fetchedMsg:
		m.loading = false
		if msg.err != nil {
			return m, func() tea.Msg { return uimsg.Err{Err: msg.err} }
		}
		m.quota, m.status, m.models, m.tools = msg.quota, msg.status, msg.models, msg.tools
		m.syncBars()
		return m, nil

	case tea.KeyPressMsg:
		if msg.String() == "r" {
			m.loading = true
			return m, m.fetch()
		}
	}
	return m, nil
}

// syncBars keeps one progress.Model per quota limit, reused across refreshes
// so the gradient fill doesn't get rebuilt every 30s.
func (m *Model) syncBars() {
	if m.quota == nil {
		return
	}
	limits := m.quota.Data.Limits
	for len(m.bars) < len(limits) {
		m.bars = append(m.bars, progress.New(progress.WithWidth(30)))
	}
	m.bars = m.bars[:len(limits)]
}

func (m Model) View() tea.View {
	if m.quota == nil {
		return tea.NewView("loading quota…")
	}

	left := m.renderQuotaPanel()
	right := m.renderHeatmapPanel()

	var body string
	if m.width >= twoColumnMinWidth {
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, left, "", right)
	}

	if m.loading {
		body += "\nrefreshing…"
	}
	return tea.NewView(body)
}

func (m Model) renderQuotaPanel() string {
	body := uistyle.SectionTitle.Render("Quota") + "\n"
	for i, limit := range m.quota.Data.Limits {
		bar := ""
		if i < len(m.bars) {
			bar = m.bars[i].ViewAs(limit.Percentage / 100)
		}
		body += fmt.Sprintf("%s\n%s %5.1f%%  (remaining %s)\n",
			limit.WindowDescription(), bar, limit.Percentage, usageview.FormatCount(int64(limit.Remaining)))
		if limit.IsTokenLimit() {
			if start := limit.WindowStart(); !start.IsZero() {
				if pace, ok := usageview.Pace(limit.Percentage/100, start, limit.ResetTime(), time.Now()); ok {
					body += uistyle.Subtle.Render(usageview.FormatPace(pace)) + "\n"
				}
			}
		}
		body += "\n"
	}
	if m.status != nil {
		body += fmt.Sprintf("Account: %s\n", m.status.Message)
	}
	return body
}

func (m Model) renderHeatmapPanel() string {
	var body string
	if m.models != nil && len(m.models.Data.ModelDataList) > 0 {
		body += uistyle.SectionTitle.Render("Model token usage (last 14d)") + "\n"
		for _, series := range m.models.Data.ModelDataList {
			body += fmt.Sprintf("  %-20s %s  %s tokens\n", series.ModelName, renderHeatmapRow(series.TokensUsage), usageview.FormatCount(series.TotalTokens))
		}
		body += fmt.Sprintf("  Total: %s calls, %s tokens\n\n",
			usageview.FormatCount(m.models.Data.TotalUsage.TotalModelCallCount),
			usageview.FormatCount(m.models.Data.TotalUsage.TotalTokensUsage))
	}

	if m.tools != nil && len(m.tools.Data.ToolDataList) > 0 {
		body += uistyle.SectionTitle.Render("Tool usage (last 14d)") + "\n"
		for _, series := range m.tools.Data.ToolDataList {
			body += fmt.Sprintf("  %-20s %s  %s calls\n", series.ToolName, renderHeatmapRow(series.UsageCount), usageview.FormatCount(series.TotalUsageCount))
		}
	}
	return body
}

// ShortHelp implements the root model's helpProvider interface.
func (m Model) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}
