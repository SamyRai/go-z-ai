// Package modelpicker implements the Chat tab's model picker overlay: a
// filterable list of available Z.AI models, fetched on open, that lets the
// user switch the model the Chat tab sends to. It is a self-contained
// tea.Model rendered as an overlay card by the root model; the chosen model
// id is returned via a Picked message the root forwards to the chat screen.
package modelpicker

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/sahilm/fuzzy"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

// Picked carries the chosen model id up to the root, which forwards it to the
// chat screen. Kept in uimsg-style here (defined as a concrete msg type the
// root recognizes) so the picker doesn't import the chat package.
type Picked struct {
	Model string
}

type fetchedMsg struct {
	models []client.ModelDetails
	err    error
}

// Model is the picker overlay. Construct via New with the API client (so it
// can fetch the catalog on open) and the currently-selected model id (to
// highlight it in the list).
type Model struct {
	client    *client.Client
	current   string
	input     textinput.Model
	models    []client.ModelDetails
	matched   []client.ModelDetails
	cursor    int
	loading   bool
	loadError string
}

// New builds the picker. The fetch is kicked off by Init (returned as a Cmd),
// so construction itself is cheap and side-effect-free.
func New(c *client.Client, current string) Model {
	in := textinput.New()
	in.Placeholder = "filter models…"
	in.Prompt = "> "
	in.Focus()
	m := Model{client: c, current: current, input: in, loading: true}
	m.matched = nil
	return m
}

func (m Model) Init() tea.Cmd {
	c := m.client
	return func() tea.Msg {
		info, err := c.Models().List(context.Background())
		if err != nil {
			return fetchedMsg{err: err}
		}
		return fetchedMsg{models: info.Models}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case fetchedMsg:
		m.loading = false
		if msg.err != nil {
			m.loadError = msg.err.Error()
			return m, nil
		}
		// Sort by id for a stable list.
		m.models = append([]client.ModelDetails(nil), msg.models...)
		sort.Slice(m.models, func(i, j int) bool { return m.models[i].ID < m.models[j].ID })
		m.applyFilter("")
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return uimsg.CloseOverlay{} }
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "tab":
			if m.cursor < len(m.matched)-1 {
				m.cursor++
			}
			return m, nil
		case "enter":
			if m.cursor >= 0 && m.cursor < len(m.matched) {
				chosen := m.matched[m.cursor].ID
				return m, func() tea.Msg { return Picked{Model: chosen} }
			}
			return m, func() tea.Msg { return uimsg.CloseOverlay{} }
		}
		// Any other key edits the filter query.
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.applyFilter(m.input.Value())
		return m, cmd
	}
	return m, nil
}

// applyFilter narrows the model list by fuzzy match over id + capabilities,
// best-score-first. An empty query keeps the sorted id order (which keeps the
// cursor position stable while the user is scanning).
func (m *Model) applyFilter(query string) {
	q := strings.TrimSpace(query)
	if q == "" {
		m.matched = append([]client.ModelDetails(nil), m.models...)
		m.cursor = 0
		// Park the cursor on the currently-selected model if present, so the
		// user can hit enter immediately to keep what they have.
		for i, md := range m.matched {
			if md.ID == m.current {
				m.cursor = i
				break
			}
		}
		return
	}
	hay := make([]string, len(m.models))
	for i, md := range m.models {
		h := md.ID
		for _, c := range md.Capabilities {
			h += " " + c
		}
		hay[i] = h
	}
	matches := fuzzy.Find(q, hay)
	type pair struct {
		md    client.ModelDetails
		score int
	}
	scored := make([]pair, 0, len(matches))
	for _, mt := range matches {
		scored = append(scored, pair{md: m.models[mt.Index], score: mt.Score})
	}
	sort.SliceStable(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
	out := make([]client.ModelDetails, 0, len(scored))
	for _, s := range scored {
		out = append(out, s.md)
	}
	m.matched = out
	if m.cursor >= len(m.matched) {
		m.cursor = max(0, len(m.matched)-1)
	}
}

func (m Model) View() tea.View {
	var b strings.Builder
	if m.loading {
		b.WriteString(uistyle.Subtle.Render("loading models…"))
	} else if m.loadError != "" {
		b.WriteString("error: " + m.loadError)
	} else if len(m.matched) == 0 {
		b.WriteString(uistyle.Subtle.Render("no models match"))
	} else {
		b.WriteString(m.input.View())
		b.WriteByte('\n')
		for i, md := range m.matched {
			mark := "  "
			if md.ID == m.current {
				mark = "● " // current model
			}
			if i == m.cursor {
				mark = "▸ "
				if md.ID == m.current {
					mark = "▸●"
				}
			}
			line := fmt.Sprintf("%s %-28s ctx %-8s %s", mark, md.ID, formatContext(md.ContextSize), joinCaps(md.Capabilities))
			b.WriteString(strings.TrimRight(line, " "))
			b.WriteByte('\n')
		}
	}
	body := strings.TrimRight(b.String(), "\n")
	return tea.NewView(uistyle.RenderOverlayCard("Switch chat model", body))
}

func formatContext(n int) string {
	if n <= 0 {
		return "—"
	}
	if n >= 1000 {
		return fmt.Sprintf("%dk", n/1000)
	}
	return fmt.Sprintf("%d", n)
}

func joinCaps(caps []string) string {
	if len(caps) == 0 {
		return ""
	}
	return strings.Join(caps, ",")
}
