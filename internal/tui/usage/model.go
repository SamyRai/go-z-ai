// Package usage implements the TUI's Usage tab: a live GLM Coding Plan quota
// and token/tool usage dashboard, backed by the same QuotaService the
// "go-z-ai usage" and "accounts quota/usage" commands use, and rendered
// through usageview's shared summaries. It refreshes with a routed tea.Tick
// and only ever calls the free monitor endpoints.
package usage

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
	"github.com/SamyRai/go-z-ai/internal/usageview"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

const (
	refreshInterval = 30 * time.Second
	// twoColumnMinWidth is the terminal width below which the quota panel
	// and the heatmap panel stack vertically instead of side by side.
	twoColumnMinWidth = 100
	// compactWidth is the width below which the heatmap panel collapses its
	// per-series ramp rows into a one-line-per-section summary, since the
	// ramps need ~45 cols each and wrap badly below this.
	compactWidth = 70
)

type tickMsg time.Time

type fetchedMsg struct {
	quota  *client.QuotaLimitResponse
	models *client.ModelUsageResponse
	tools  *client.ToolUsageResponse
	err    error
}

// Model is the Usage tab's screen model.
type Model struct {
	client  func() *client.Client
	selfTab int           // this screen's tab index, used to route results back
	every   time.Duration // auto-refresh interval

	quota  *client.QuotaLimitResponse
	models *client.ModelUsageResponse
	tools  *client.ToolUsageResponse
	bars   []progress.Model // one per m.quota.Data.Limits entry
	err    error

	loading bool
	width   int
}

// New builds the Usage screen. c returns the current API client; selfTab is
// this screen's tab index in the root model.
func New(c func() *client.Client, selfTab int) Model {
	return Model{client: c, selfTab: selfTab, every: refreshInterval}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(uimsg.Route(m.selfTab, m.fetch()), m.tick())
}

// tick schedules the next refresh, routed so it reaches this tab even while
// another one is active.
func (m Model) tick() tea.Cmd {
	return uimsg.Route(m.selfTab, tea.Tick(m.every, func(t time.Time) tea.Msg { return tickMsg(t) }))
}

func (m Model) fetch() tea.Cmd {
	c := m.client()
	return func() tea.Msg {
		ctx := context.Background()
		quota, err := c.Quota().GetQuotaLimit(ctx)
		if err != nil {
			return fetchedMsg{err: err}
		}
		out := fetchedMsg{quota: quota}
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
		return m, tea.Batch(uimsg.Route(m.selfTab, m.fetch()), m.tick())

	case fetchedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.quota, m.models, m.tools = msg.quota, msg.models, msg.tools
			m.syncBars()
		}
		return m, nil

	case tea.KeyPressMsg:
		if msg.String() == "r" {
			return m.Refresh()
		}
	}
	return m, nil
}

// Refresh forces an immediate re-fetch. Implements the root model's
// refresher interface (the palette's "Refresh current tab" and the 'r' key).
func (m Model) Refresh() (tea.Model, tea.Cmd) {
	m.loading = true
	return m, uimsg.Route(m.selfTab, m.fetch())
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
	if m.quota == nil && m.err != nil {
		body := uistyle.EmptyTitle.Render("No coding-plan quota") + "\n\n" +
			uistyle.EmptyHint.Render(m.err.Error()) + "\n\n" +
			uistyle.Subtle.Render("Quota and usage are available for GLM Coding Plan accounts. Press 'r' to retry.")
		return tea.NewView(body)
	}
	if m.quota == nil {
		// Skeleton: a title + a few dimmed bar rows so the layout reads as
		// "filling in" instead of a bare text label before first data.
		body := uistyle.SectionTitle.Render("Quota") + "\n"
		for range 3 {
			body += uistyle.SkeletonRow(28) + "\n" +
				uistyle.SkeletonRow(28) + "\n\n"
		}
		return tea.NewView(body)
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
	body := uistyle.SectionTitle.Render(fmt.Sprintf("Quota · %s plan", m.quota.Data.Level)) + "\n"
	mode := usageview.BillingModeOf(m.quota.Data.Limits)
	now := time.Now()
	for i, limit := range m.quota.Data.Limits {
		s := usageview.SummarizeLimit(limit, mode, now, time.Local)
		bar := ""
		if i < len(m.bars) {
			bar = m.bars[i].ViewAs(min(s.Used, 1))
		}
		body += fmt.Sprintf("%s\n%s %5.1f%%\n", s.Title, bar, s.Used*100)
		for _, line := range []string{s.Counts, s.Reset, s.Pace} {
			if line != "" {
				body += uistyle.Subtle.Render(line) + "\n"
			}
		}
		for _, tool := range s.Tools {
			body += uistyle.Subtle.Render("  "+tool) + "\n"
		}
		if s.Peak != "" {
			body += uistyle.ToastWarn.Render(s.Peak) + "\n"
		}
		body += "\n"
	}
	if m.err != nil {
		body += uistyle.ToastWarn.Render("refresh failed: "+m.err.Error()) + "\n"
	}
	return body
}

func (m Model) renderHeatmapPanel() string {
	var body string
	hasModels := m.models != nil && len(m.models.Data.ModelDataList) > 0
	hasTools := m.tools != nil && len(m.tools.Data.ToolDataList) > 0

	// Compact mode on narrow terminals: drop the per-series ramp rows (which
	// need ~45 cols each and wrap badly below ~70) and show one summary line
	// per section with totals only. The full ramp view returns above the
	// threshold.
	compact := m.width > 0 && m.width < compactWidth

	if hasModels {
		body += uistyle.SectionTitle.Render("Model token usage (last 14d)") + "\n"
		if compact {
			body += fmt.Sprintf("  %s models · %s calls · %s tokens\n",
				usageview.FormatCount(int64(len(m.models.Data.ModelDataList))),
				usageview.FormatCount(m.models.Data.TotalUsage.TotalModelCallCount),
				usageview.FormatCount(m.models.Data.TotalUsage.TotalTokensUsage))
		} else {
			for _, series := range m.models.Data.ModelDataList {
				body += fmt.Sprintf("  %-20s %s  %s tokens\n", series.ModelName, renderHeatmapRow(series.TokensUsage), usageview.FormatCount(series.TotalTokens))
			}
			body += fmt.Sprintf("  Total: %s calls, %s tokens\n\n",
				usageview.FormatCount(m.models.Data.TotalUsage.TotalModelCallCount),
				usageview.FormatCount(m.models.Data.TotalUsage.TotalTokensUsage))
		}
	}

	if hasTools {
		body += uistyle.SectionTitle.Render("Tool usage (last 14d)") + "\n"
		if compact {
			// Sum the per-series counts (the tool total struct has no single
			// aggregate field, only per-tool-type counts).
			var total int64
			for _, series := range m.tools.Data.ToolDataList {
				total += series.TotalUsageCount
			}
			body += fmt.Sprintf("  %s tools · %s total calls\n",
				usageview.FormatCount(int64(len(m.tools.Data.ToolDataList))),
				usageview.FormatCount(total))
		} else {
			for _, series := range m.tools.Data.ToolDataList {
				body += fmt.Sprintf("  %-20s %s  %s calls\n", series.ToolName, renderHeatmapRow(series.UsageCount), usageview.FormatCount(series.TotalUsageCount))
			}
		}
	}

	// If both sections came back empty (loaded successfully but no usage in the
	// window), say so instead of silently rendering nothing — a blank panel
	// reads as "broken", not "no data".
	if !hasModels && !hasTools {
		body += uistyle.Subtle.Render("no model or tool usage in the last 14 days") + "\n"
	}
	return body
}

// ShortHelp implements the root model's helpProvider interface.
func (m Model) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}
