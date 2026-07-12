// Package media implements the TUI's Media tab: image generation, video
// generation, audio transcription, and OCR/layout parsing over pkg/client's
// ImagesService/VideosService/AudioService/LayoutService — the same
// services the "zai-client image/video/audio/ocr" commands use.
package media

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/SamyRai/go-z-ai/pkg/tui/uistyle"
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

const pollInterval = 3 * time.Second

type resultMsg struct {
	text    string
	err     error
	videoID string // set when a video task was submitted; triggers polling
}

type pollMsg struct{ id string }

// Model is the Media tab's screen model.
type Model struct {
	client *client.Client
	active form
	inputs [formCount]textinput.Model
	result viewport.Model
	spin   spinner.Model
	busy   bool

	pollingVideoID string
}

// New builds the Media screen. c must be non-nil.
func New(c *client.Client) Model {
	image := textinput.New()
	image.Placeholder = "image prompt"
	video := textinput.New()
	video.Placeholder = "video prompt"
	audio := textinput.New()
	audio.Placeholder = "path to .wav/.mp3 file"
	ocr := textinput.New()
	ocr.Placeholder = "path to file, or a URL"

	return Model{
		client: c,
		inputs: [formCount]textinput.Model{formImage: image, formVideo: video, formAudio: audio, formOCR: ocr},
		result: viewport.New(),
		spin:   spinner.New(),
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
		if msg.err != nil {
			m.busy = false
			m.pollingVideoID = ""
			m.result.SetContent("error: " + msg.err.Error())
			return m, nil
		}
		m.result.SetContent(msg.text)
		if msg.videoID != "" {
			m.pollingVideoID = msg.videoID
			return m, pollAfter(msg.videoID)
		}
		// Terminal result: also stop the video-poll spinner if one was live.
		m.busy = false
		m.pollingVideoID = ""
		return m, nil

	case pollMsg:
		return m, m.checkVideo(msg.id)

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

func pollAfter(id string) tea.Cmd {
	return tea.Tick(pollInterval, func(time.Time) tea.Msg { return pollMsg{id: id} })
}

func (m Model) checkVideo(id string) tea.Cmd {
	c := m.client
	return func() tea.Msg {
		result, err := c.GetAsyncResult(context.Background(), id)
		if err != nil {
			return resultMsg{err: err}
		}
		switch result.TaskStatus {
		case client.TaskStatusSuccess:
			text := fmt.Sprintf("status: %s\n", result.TaskStatus)
			for i, v := range result.VideoResult {
				text += fmt.Sprintf("video %d: %s\n", i+1, v.URL)
			}
			return resultMsg{text: text}
		case client.TaskStatusFail:
			return resultMsg{err: fmt.Errorf("video generation failed")}
		default:
			return resultMsg{text: fmt.Sprintf("status: %s (polling every %s)", result.TaskStatus, pollInterval), videoID: id}
		}
	}
}

func (m Model) submit() tea.Cmd {
	c := m.client
	switch m.active {
	case formImage:
		prompt := m.inputs[formImage].Value()
		return func() tea.Msg {
			resp, err := c.Images().Generate(context.Background(), client.ImageGenerationRequest{Model: "glm-image", Prompt: prompt})
			if err != nil {
				return resultMsg{err: err}
			}
			text := "open in browser:\n"
			for _, d := range resp.Data {
				text += d.URL + "\n"
			}
			return resultMsg{text: text}
		}

	case formVideo:
		prompt := m.inputs[formVideo].Value()
		return func() tea.Msg {
			resp, err := c.Videos().Generate(context.Background(), client.VideoGenerationRequest{Model: "cogvideox-3", Prompt: prompt})
			if err != nil {
				return resultMsg{err: err}
			}
			return resultMsg{
				text:    fmt.Sprintf("status: %s (polling every %s)", resp.TaskStatus, pollInterval),
				videoID: resp.ID,
			}
		}

	case formAudio:
		path := m.inputs[formAudio].Value()
		return func() tea.Msg {
			data, err := os.ReadFile(path)
			if err != nil {
				return resultMsg{err: err}
			}
			resp, err := c.Audio().Transcribe(context.Background(), client.AudioTranscriptionRequest{
				FileName: filepath.Base(path),
				FileData: data,
			})
			if err != nil {
				return resultMsg{err: err}
			}
			return resultMsg{text: resp.Text}
		}

	default: // formOCR
		target := m.inputs[formOCR].Value()
		return func() tea.Msg {
			// The layout API takes URLs verbatim but wants local files as
			// base64 — same handling as the "ocr parse" CLI command.
			file := target
			if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
				data, err := os.ReadFile(target)
				if err != nil {
					return resultMsg{err: err}
				}
				file = base64.StdEncoding.EncodeToString(data)
			}
			resp, err := c.Layout().Parse(context.Background(), client.LayoutParsingRequest{File: file})
			if err != nil {
				return resultMsg{err: err}
			}
			return resultMsg{text: resp.MDResults}
		}
	}
}

func (m Model) View() tea.View {
	body := uistyle.RenderPills(int(m.active), formNames[:]) + "\n\n"
	body += m.inputs[m.active].View() + "\n"
	body += "\n" + m.result.View()
	if m.busy || m.pollingVideoID != "" {
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
