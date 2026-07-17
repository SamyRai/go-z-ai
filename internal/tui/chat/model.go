// Package chat implements the TUI's Chat tab: a streaming conversation over
// pkg/client's ChatService, the same service "zai-client chat" uses.
package chat

import (
	"context"
	"errors"
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

const defaultModel = "glm-5.2"

// Model is the Chat tab's screen model.
type Model struct {
	client   *client.Client
	input    textarea.Model
	view     viewport.Model
	spin     spinner.Model
	messages []client.Message
	model    string

	renderer      *glamour.TermRenderer
	rendererWidth int
	rendererDark  bool

	streaming bool
	handle    streamHandle
	pending   string // partial assistant reply accumulated mid-stream
}

// New builds the Chat screen. c must be non-nil.
func New(c *client.Client) Model {
	in := textarea.New()
	in.Placeholder = "Type a message, ctrl+s to send…"
	in.Focus()

	return Model{
		client: c,
		input:  in,
		view:   viewport.New(),
		spin:   spinner.New(),
		model:  defaultModel,
	}
}

// Streaming reports whether a request is in flight, so the root model can
// route ctrl+c to cancel-in-place instead of quitting the whole program.
func (m Model) Streaming() bool { return m.streaming }

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
		m.view.SetContent(m.transcript())
		return m, nil

	case tea.BackgroundColorMsg:
		m.ensureRenderer(m.rendererWidth, msg.IsDark())
		return m, nil

	case chunkMsg:
		for _, choice := range msg.Choices {
			m.pending += choice.Delta.Content
		}
		m.view.SetContent(m.transcript())
		m.view.GotoBottom()
		return m, waitForChunk(m.handle)

	case streamDoneMsg:
		m.streaming = false
		if m.pending != "" {
			m.messages = append(m.messages, client.Message{Role: "assistant", Content: m.pending})
			m.pending = ""
		}
		m.view.SetContent(m.transcript())
		m.view.GotoBottom()
		if msg.err != nil && !errors.Is(msg.err, context.Canceled) {
			return m, func() tea.Msg { return uimsg.Err{Err: msg.err} }
		}
		return m, nil

	case spinner.TickMsg:
		if !m.streaming {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.streaming {
				m.handle.cancel()
				return m, nil
			}
		case "ctrl+s":
			if !m.streaming && m.input.Value() != "" {
				return m.send()
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) send() (tea.Model, tea.Cmd) {
	text := m.input.Value()
	m.messages = append(m.messages, client.Message{Role: "user", Content: text})
	m.input.Reset()
	m.streaming = true
	m.pending = ""
	m.view.SetContent(m.transcript())
	m.view.GotoBottom()

	req := client.ChatRequest{
		Model:       m.model,
		Messages:    m.messages,
		Temperature: 0.7,
		TopP:        0.95,
		MaxTokens:   4096,
	}
	cmd, handle := startStream(m.client, req)
	m.handle = handle
	return m, tea.Batch(cmd, m.spin.Tick)
}

func (m Model) transcript() string {
	var out string
	for _, msg := range m.messages {
		if msg.Role == "assistant" {
			out += "assistant:\n" + m.renderMarkdown(msg.Content) + "\n"
		} else {
			out += fmt.Sprintf("%s: %s\n\n", msg.Role, msg.Content)
		}
	}
	if m.pending != "" {
		out += "assistant:\n" + m.renderMarkdown(m.pending)
	}
	return out
}

func (m Model) View() tea.View {
	body := m.view.View() + "\n" + m.input.View()
	if m.streaming {
		body += "\n" + m.spin.View() + " streaming… (ctrl+c to cancel)"
	} else {
		body += "\nctrl+s: send"
	}
	return tea.NewView(body)
}

// ShortHelp implements the root model's helpProvider interface.
func (m Model) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "send")),
		key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "cancel stream")),
	}
}
