// Package formtab is the screen shared by the Media and Tools tabs: a row of
// single-input request forms (switched with ↑/↓), one in-flight request at a
// time (cancellable with esc), and a scrollable result pane. Tabs supply only
// their Forms.
package formtab

import (
	"context"
	"errors"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

// Form is one request form.
type Form struct {
	Name        string
	Placeholder string
	// Run performs the request for the entered value and renders its
	// result. It must honor ctx, which esc cancels.
	Run func(ctx context.Context, c *client.Client, value string) (string, error)
}

type resultMsg struct {
	text string
	err  error
}

// Model is a form-tab screen.
type Model struct {
	client  func() *client.Client
	selfTab int // results are routed back here
	forms   []Form
	names   []string
	inputs  []textinput.Model
	active  int
	result  viewport.Model
	spin    spinner.Model
	busy    bool
	cancel  context.CancelFunc // aborts the in-flight request; nil when idle
}

// New builds a form tab. c returns the current API client; selfTab is the
// screen's tab index in the root model.
func New(c func() *client.Client, selfTab int, forms ...Form) Model {
	m := Model{client: c, selfTab: selfTab, forms: forms, result: viewport.New(), spin: spinner.New()}
	for _, f := range forms {
		in := textinput.New()
		in.Placeholder = f.Placeholder
		m.inputs = append(m.inputs, in)
		m.names = append(m.names, f.Name)
	}
	m.inputs[0].Focus()
	return m
}

func (m Model) Init() tea.Cmd { return nil }

// CapturesInput is always true: the active form's input has focus.
func (m Model) CapturesInput() bool { return true }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.result.SetWidth(msg.Width)
		m.result.SetHeight(max(msg.Height-6, 3))
		for i := range m.inputs {
			m.inputs[i].SetWidth(max(msg.Width-4, 10))
		}
		return m, nil

	case resultMsg:
		m.busy, m.cancel = false, nil
		switch {
		case errors.Is(msg.err, context.Canceled):
			// User-initiated; the esc handler already said so.
		case msg.err != nil:
			m.result.SetContent("error: " + msg.err.Error())
		default:
			m.result.SetContent(msg.text)
		}
		return m, nil

	case spinner.TickMsg:
		if !m.busy {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		// tab/shift+tab belong to the root's tab bar, so forms switch on
		// up/down, which a single-line input doesn't use.
		// focus and submit mutate m, so they run before m is returned (Go
		// leaves the order of `return m, m.f()` unspecified).
		var cmd tea.Cmd
		switch msg.String() {
		case "down":
			cmd = m.focus((m.active + 1) % len(m.forms))
		case "up":
			cmd = m.focus((m.active + len(m.forms) - 1) % len(m.forms))
		case "esc":
			if m.cancel != nil {
				m.cancel()
				m.result.SetContent("cancelled")
			}
		case "enter":
			if !m.busy {
				cmd = m.submit()
			}
		default:
			m.inputs[m.active], cmd = m.inputs[m.active].Update(msg)
		}
		return m, cmd
	}

	var cmd tea.Cmd
	m.inputs[m.active], cmd = m.inputs[m.active].Update(msg)
	return m, cmd
}

// focus moves keyboard focus to form i.
func (m *Model) focus(i int) tea.Cmd {
	m.inputs[m.active].Blur()
	m.active = i
	return m.inputs[i].Focus()
}

// submit runs the active form on a cancellable context, routing the result
// back to this tab however the user moves between tabs meanwhile.
func (m *Model) submit() tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.busy, m.cancel = true, cancel
	run, value, c := m.forms[m.active].Run, m.inputs[m.active].Value(), m.client()
	request := func() tea.Msg {
		text, err := run(ctx, c, value)
		return resultMsg{text: text, err: err}
	}
	return tea.Batch(uimsg.Route(m.selfTab, request), m.spin.Tick)
}

func (m Model) View() tea.View {
	body := uistyle.RenderPills(m.active, m.names) + "\n\n" +
		m.inputs[m.active].View() + "\n\n" + m.result.View()
	if m.busy {
		body += "\n" + m.spin.View() + " working… (esc to cancel)"
	}
	return tea.NewView(body)
}

// ShortHelp implements the root model's helpProvider interface.
func (m Model) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑/↓", "switch form")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	}
}
