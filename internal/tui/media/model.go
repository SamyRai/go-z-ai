// Package media implements the TUI's Media tab: image generation, video
// generation, audio transcription, and OCR/layout parsing over pkg/client's
// ImagesService/VideosService/AudioService/LayoutService — the same
// services the "zai-client image/video/audio/ocr" commands use.
package media

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/fileinput"
	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

type form int

const (
	formImage form = iota
	formVideo
	formAudio
	formOCR
	formCount
)

var formNames = [...]string{"Image", "Video", "Audio", "OCR"}

type resultMsg struct {
	text string
	err  error
}

// Model is the Media tab's screen model.
type Model struct {
	client  *client.Client
	selfTab int // this screen's tab index, used to route async results back
	active  form
	inputs  [formCount]textinput.Model
	result  viewport.Model
	spin    spinner.Model
	busy    bool

	// cancel aborts the in-flight submission (a video generation can poll for
	// minutes); nil when nothing is running.
	cancel context.CancelFunc
}

// New builds the Media screen. c must be non-nil. selfTab is this screen's tab
// index in the root model, so async results can be routed back here even when
// the user has switched to another tab.
func New(c *client.Client, selfTab int) Model {
	image := textinput.New()
	image.Placeholder = "image prompt"
	video := textinput.New()
	video.Placeholder = "video prompt"
	audio := textinput.New()
	audio.Placeholder = "path to .wav/.mp3 file"
	ocr := textinput.New()
	ocr.Placeholder = "path to file, or a URL"

	return Model{
		client:  c,
		selfTab: selfTab,
		inputs:  [formCount]textinput.Model{formImage: image, formVideo: video, formAudio: audio, formOCR: ocr},
		result:  viewport.New(),
		spin:    spinner.New(),
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
		m.cancel = nil
		if msg.err != nil {
			// A cancellation is user-initiated, not a failure — leave the
			// "cancelled" notice the esc handler already set.
			if errors.Is(msg.err, context.Canceled) {
				return m, nil
			}
			m.result.SetContent("error: " + msg.err.Error())
			return m, nil
		}
		m.result.SetContent(msg.text)
		return m, nil

	case spinner.TickMsg:
		if !m.busy {
			return m, nil
		}
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
		case "esc":
			if m.busy && m.cancel != nil {
				m.cancel()
				m.result.SetContent("cancelled")
			}
			return m, nil
		case "enter":
			if m.busy {
				return m, nil
			}
			m.busy = true
			// Call submit before the return so it records the cancel func on
			// m before m is copied into the return value.
			cmd := m.submit()
			return m, tea.Batch(cmd, m.spin.Tick)
		}
	}

	var cmd tea.Cmd
	m.inputs[m.active], cmd = m.inputs[m.active].Update(msg)
	return m, cmd
}

// submit starts the active form's operation on a background context (stored on
// the model so esc can cancel it) and returns a single tea.Cmd. The Cmd runs
// the whole operation — including a video task's poll-to-completion via
// WaitForResult — and wraps its terminal result in uimsg.Routed so the result
// reaches this screen even if the user has switched tabs meanwhile. It has a
// pointer receiver because it records the cancel func on the model.
func (m *Model) submit() tea.Cmd {
	c := m.client
	self := m.selfTab
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	// routed wraps a resultMsg for delivery back to this screen.
	routed := func(r resultMsg) tea.Msg { return uimsg.Routed{Tab: self, Msg: r} }

	switch m.active {
	case formImage:
		prompt := m.inputs[formImage].Value()
		return func() tea.Msg {
			resp, err := c.Images().Generate(ctx, client.ImageGenerationRequest{Model: "glm-image", Prompt: prompt})
			if err != nil {
				return routed(resultMsg{err: err})
			}
			text := "open in browser:\n"
			for _, d := range resp.Data {
				text += d.URL + "\n"
			}
			return routed(resultMsg{text: text})
		}

	case formVideo:
		prompt := m.inputs[formVideo].Value()
		return func() tea.Msg {
			resp, err := c.Videos().Generate(ctx, client.VideoGenerationRequest{Model: "cogvideox-3", Prompt: prompt})
			if err != nil {
				return routed(resultMsg{err: err})
			}
			// Video is always async; WaitForResult polls to a terminal state
			// (or until ctx is cancelled) inside this one Cmd, so there's no
			// multi-message poll loop for a tab switch to strand.
			result, err := c.WaitForResult(ctx, resp.ID, 0)
			if err != nil {
				return routed(resultMsg{err: err})
			}
			if result.TaskStatus == client.TaskStatusFail {
				return routed(resultMsg{err: fmt.Errorf("video generation failed")})
			}
			text := fmt.Sprintf("status: %s\n", result.TaskStatus)
			for i, v := range result.VideoResult {
				text += fmt.Sprintf("video %d: %s\n", i+1, v.URL)
			}
			return routed(resultMsg{text: text})
		}

	case formAudio:
		path := m.inputs[formAudio].Value()
		return func() tea.Msg {
			data, err := os.ReadFile(path)
			if err != nil {
				return routed(resultMsg{err: err})
			}
			resp, err := c.Audio().Transcribe(ctx, client.AudioTranscriptionRequest{
				FileName: filepath.Base(path),
				FileData: data,
			})
			if err != nil {
				return routed(resultMsg{err: err})
			}
			return routed(resultMsg{text: resp.Text})
		}

	default: // formOCR
		target := m.inputs[formOCR].Value()
		return func() tea.Msg {
			file, err := fileinput.FileOrURL(target)
			if err != nil {
				return routed(resultMsg{err: err})
			}
			resp, err := c.Layout().Parse(ctx, client.LayoutParsingRequest{File: file})
			if err != nil {
				return routed(resultMsg{err: err})
			}
			return routed(resultMsg{text: resp.MDResults})
		}
	}
}

func (m Model) View() tea.View {
	body := uistyle.RenderPills(int(m.active), formNames[:]) + "\n\n"
	body += m.inputs[m.active].View() + "\n"
	body += "\n" + m.result.View()
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
