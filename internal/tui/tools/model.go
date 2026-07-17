// Package tools implements the TUI's Tools tab: three independent
// request/response forms (web search, web reader, tokenizer) over
// pkg/client's ToolsService, the same service the "zai-client tools"
// commands use.
package tools

import (
	"context"
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

type form int

const (
	formSearch form = iota
	formReader
	formTokenize
	formCount
)

var formNames = [...]string{"Web Search", "Web Reader", "Tokenizer"}

type resultMsg struct {
	text string
	err  error
}

// Model is the Tools tab's screen model.
type Model struct {
	client *client.Client
	active form
	inputs [formCount]textinput.Model
	result viewport.Model
	spin   spinner.Model
	busy   bool
}

// New builds the Tools screen. c must be non-nil.
func New(c *client.Client) Model {
	search := textinput.New()
	search.Placeholder = "search query"
	reader := textinput.New()
	reader.Placeholder = "https://..."
	tokenize := textinput.New()
	tokenize.Placeholder = "text to tokenize"

	vp := viewport.New()
	sp := spinner.New()

	return Model{
		client: c,
		inputs: [formCount]textinput.Model{formSearch: search, formReader: reader, formTokenize: tokenize},
		result: vp,
		spin:   sp,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.result.SetWidth(msg.Width)
		m.result.SetHeight(max(msg.Height-6, 3))
		return m, nil

	case resultMsg:
		m.busy = false
		if msg.err != nil {
			m.result.SetContent("error: " + msg.err.Error())
		} else {
			m.result.SetContent(msg.text)
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		// Forms cycle on up/down: tab/shift+tab never reach this screen (the
		// root model consumes them for tab-bar navigation), and up/down are
		// not bound by the single-line textinput.
		switch msg.String() {
		case "down":
			m.active = (m.active + 1) % formCount
			return m, nil
		case "up":
			m.active = (m.active + formCount - 1) % formCount
			return m, nil
		case "enter":
			if m.busy {
				return m, nil
			}
			m.busy = true
			return m, tea.Batch(m.submit(), m.spin.Tick)
		}
	}

	var cmd tea.Cmd
	m.inputs[m.active], cmd = m.inputs[m.active].Update(msg)
	return m, cmd
}

func (m Model) submit() tea.Cmd {
	c := m.client
	switch m.active {
	case formSearch:
		q := m.inputs[formSearch].Value()
		return func() tea.Msg {
			resp, err := c.Tools().WebSearch(context.Background(), client.WebSearchRequest{
				SearchQuery:  q,
				SearchEngine: client.SearchEnginePro,
			})
			if err != nil {
				return resultMsg{err: err}
			}
			var text string
			for _, r := range resp.SearchResult {
				text += fmt.Sprintf("%s\n%s\n\n", r.Title, r.Link)
			}
			return resultMsg{text: text}
		}
	case formReader:
		url := m.inputs[formReader].Value()
		return func() tea.Msg {
			resp, err := c.Tools().WebReader(context.Background(), client.WebReaderRequest{URL: url, WithImagesSummary: true})
			if err != nil {
				return resultMsg{err: err}
			}
			if resp.ReaderResult == nil {
				return resultMsg{err: fmt.Errorf("empty response (id %s)", resp.ID)}
			}
			return resultMsg{text: resp.ReaderResult.Content}
		}
	default:
		text := m.inputs[formTokenize].Value()
		return func() tea.Msg {
			resp, err := c.Tools().Tokenize(context.Background(), client.TokenizerRequest{
				Model:    "glm-4.7",
				Messages: []client.Message{{Role: "user", Content: text}},
			})
			if err != nil {
				return resultMsg{err: err}
			}
			if resp.Usage == nil {
				return resultMsg{err: fmt.Errorf("empty response (id %s)", resp.ID)}
			}
			return resultMsg{text: fmt.Sprintf("tokens: %d", resp.Usage.TotalTokens)}
		}
	}
}

func (m Model) View() tea.View {
	body := uistyle.RenderPills(int(m.active), formNames[:]) + "\n\n"
	body += m.inputs[m.active].View() + "\n"
	body += "\n" + m.result.View()
	if m.busy {
		body += "\n" + m.spin.View() + " working..."
	}
	return tea.NewView(body)
}

// ShortHelp implements the root model's helpProvider interface.
func (m Model) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑/↓", "switch form")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
	}
}
