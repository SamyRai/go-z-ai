// Package chat implements the TUI's Chat tab: a streaming conversation over
// pkg/client's ChatService, the same service "go-z-ai chat" uses. Completed
// messages are rendered as markdown once; the reply being streamed is shown
// as plain text, with the model's reasoning dimmed above it.
package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

// Model is the Chat tab's screen model.
type Model struct {
	client   func() *client.Client
	selfTab  int // stream chunks are routed back here
	input    textarea.Model
	view     viewport.Model
	spin     spinner.Model
	messages []client.Message
	history  string // rendered transcript of messages, rebuilt on change
	model    string

	renderer      *glamour.TermRenderer
	rendererWidth int
	rendererDark  bool

	stream           *stream // nil when idle
	pending          string  // assistant reply accumulated mid-stream
	pendingReasoning string  // reasoning accumulated mid-stream
}

// New builds the Chat screen. c returns the current API client; selfTab is
// the screen's tab index in the root model.
func New(c func() *client.Client, selfTab int) Model {
	in := textarea.New()
	in.Placeholder = "Type a message, ctrl+s to send…"
	in.Focus()

	return Model{
		client:  c,
		selfTab: selfTab,
		input:   in,
		view:    viewport.New(),
		spin:    spinner.New(),
		model:   client.DefaultModel,
	}
}

// Streaming reports whether a request is in flight, so the root model routes
// ctrl+c to cancel it instead of quitting.
func (m Model) Streaming() bool { return m.stream != nil }

// CapturesInput is always true: the message box has focus.
func (m Model) CapturesInput() bool { return true }

// Model returns the id of the model the chat tab will send to. Implements the
// root's chatModelGetter interface so the model-picker overlay can highlight
// the currently-selected entry.
func (m Model) ModelID() string { return m.model }

// SetModel changes the model id used for the next send. Called by the root
// when the model-picker overlay returns a choice. Implements the root's
// chatModelSetter interface (a narrow seam so the root needn't know the chat
// model's full type).
func (m Model) SetModel(id string) (tea.Model, tea.Cmd) {
	if id == "" {
		return m, nil
	}
	m.model = id
	m.refreshView()
	return m, nil
}

func (m Model) Init() tea.Cmd { return nil }

// ensureRenderer (re)builds the glamour renderer only when the width or the
// terminal's dark/light background changes — construction does real work
// (loads/parses a style), so it must not happen per streamed chunk.
func (m *Model) ensureRenderer(width int, dark bool) {
	if m.renderer != nil && width == m.rendererWidth && dark == m.rendererDark {
		return
	}
	style := "light"
	if dark {
		style = "dark"
	}
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(width))
	if err != nil {
		return // keep the previous renderer (or nil, falling back to plain text)
	}
	m.renderer = r
	m.rendererWidth = width
	m.rendererDark = dark
	m.renderHistory()
}

func (m Model) renderMarkdown(s string) string {
	if m.renderer == nil || s == "" {
		return s
	}
	out, err := m.renderer.Render(s)
	if err != nil {
		return s
	}
	return out
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.input.SetWidth(msg.Width)
		m.input.SetHeight(3)
		m.view.SetWidth(msg.Width)
		m.view.SetHeight(max(msg.Height-6, 3))
		m.ensureRenderer(msg.Width, m.rendererDark)
		m.refreshView()
		return m, nil

	case tea.BackgroundColorMsg:
		m.ensureRenderer(m.rendererWidth, msg.IsDark())
		m.refreshView()
		return m, nil

	case chunkMsg:
		if m.stream == nil {
			return m, nil
		}
		for _, choice := range msg.Choices {
			m.pending += choice.Delta.Content
			m.pendingReasoning += choice.Delta.ReasoningContent
		}
		m.refreshView()
		m.view.GotoBottom()
		return m, m.stream.wait()

	case streamDoneMsg:
		m.stream = nil
		if m.pending != "" || m.pendingReasoning != "" {
			// Keep the reasoning on the message so it is echoed back on the
			// next turn (preserved / interleaved thinking).
			m.messages = append(m.messages, client.Message{Role: "assistant", Content: m.pending, ReasoningContent: m.pendingReasoning})
			m.pending, m.pendingReasoning = "", ""
			m.renderHistory()
		}
		m.refreshView()
		m.view.GotoBottom()
		if msg.err != nil && !errors.Is(msg.err, context.Canceled) {
			return m, func() tea.Msg { return uimsg.Err{Err: msg.err} }
		}
		return m, nil

	case spinner.TickMsg:
		if m.stream == nil {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			// Cancelling makes the pending pull return; its streamDoneMsg
			// clears the stream.
			if m.stream != nil {
				m.stream.cancel()
			}
			return m, nil
		case "ctrl+s":
			if m.stream == nil && strings.TrimSpace(m.input.Value()) != "" {
				return m.send()
			}
			return m, nil
		case "ctrl+o":
			// The root owns the model picker; it calls SetModel on a pick.
			return m, func() tea.Msg { return uimsg.OpenModelPicker{} }
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// send starts streaming a reply to the typed message, with the model's own
// sampling defaults.
func (m Model) send() (tea.Model, tea.Cmd) {
	m.messages = append(m.messages, client.Message{Role: "user", Content: m.input.Value()})
	m.input.Reset()
	m.renderHistory()
	m.refreshView()
	m.view.GotoBottom()

	req := client.ChatRequest{Model: m.model, Messages: m.messages}
	s, cmd := startStream(m.client(), req, m.selfTab)
	m.stream = s
	return m, tea.Batch(cmd, m.spin.Tick)
}

// renderHistory re-renders the completed messages.
func (m *Model) renderHistory() {
	var b strings.Builder
	for _, msg := range m.messages {
		if msg.Role == "assistant" {
			b.WriteString(uistyle.SectionTitle.Render("assistant") + "\n" + m.renderMarkdown(msg.Content) + "\n")
		} else {
			fmt.Fprintf(&b, "%s %s\n\n", uistyle.SectionTitle.Render(msg.Role+":"), msg.Content)
		}
	}
	m.history = b.String()
}

// refreshView sets the viewport to the rendered history plus the reply in
// progress.
func (m *Model) refreshView() {
	if m.history == "" && m.stream == nil {
		m.view.SetContent(fmt.Sprintf("model: %s\n\nType a message, ctrl+s to send, ctrl+o to switch model.", m.model))
		return
	}
	out := m.history
	if m.stream != nil {
		out += uistyle.SectionTitle.Render("assistant") + "\n"
		if m.pendingReasoning != "" {
			out += uistyle.Subtle.Render(m.pendingReasoning) + "\n\n"
		}
		out += m.pending
	}
	m.view.SetContent(out)
}

func (m Model) View() tea.View {
	body := m.view.View() + "\n" + m.input.View()
	if m.stream != nil {
		body += "\n" + m.spin.View() + " streaming… (ctrl+c to cancel)"
	}
	return tea.NewView(body)
}

// ShortHelp implements the root model's helpProvider interface.
func (m Model) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "send")),
		key.NewBinding(key.WithKeys("ctrl+o"), key.WithHelp("ctrl+o", "model")),
		key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "cancel stream")),
	}
}
