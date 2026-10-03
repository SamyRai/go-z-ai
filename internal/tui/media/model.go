// Package media implements the TUI's Media tab: image and video generation,
// audio transcription, and OCR, over the same client services the
// "go-z-ai image/video/audio/ocr" commands use.
package media

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SamyRai/go-z-ai/internal/fileinput"
	"github.com/SamyRai/go-z-ai/internal/tui/formtab"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

// New builds the Media screen. c returns the current API client; selfTab is
// the screen's tab index in the root model.
func New(c func() *client.Client, selfTab int) formtab.Model {
	return formtab.New(c, selfTab,
		formtab.Form{Name: "Image", Placeholder: "image prompt", Run: generateImage},
		formtab.Form{Name: "Video", Placeholder: "video prompt", Run: generateVideo},
		formtab.Form{Name: "Audio", Placeholder: "path to a .wav/.mp3 file (≤30 s)", Run: transcribe},
		formtab.Form{Name: "OCR", Placeholder: "path to an image/PDF, or a URL", Run: parseLayout},
	)
}

func generateImage(ctx context.Context, c *client.Client, prompt string) (string, error) {
	resp, err := c.Images().Generate(ctx, client.ImageGenerationRequest{Model: client.ModelGLMImage, Prompt: prompt})
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("open in a browser (links expire in 30 days):\n")
	for _, img := range resp.Data {
		b.WriteString(img.URL + "\n")
	}
	return b.String(), nil
}

// generateVideo submits the task and polls it to completion within this one
// request, so a tab switch can't strand a poll loop.
func generateVideo(ctx context.Context, c *client.Client, prompt string) (string, error) {
	task, err := c.Videos().Generate(ctx, client.VideoGenerationRequest{Model: client.ModelCogVideoX3, Prompt: prompt})
	if err != nil {
		return "", err
	}
	result, err := c.WaitForResult(ctx, task.ID, 0)
	if err != nil {
		return "", err
	}
	if result.TaskStatus == client.TaskStatusFail {
		return "", fmt.Errorf("video generation failed (task %s)", task.ID)
	}
	var b strings.Builder
	for i, v := range result.VideoResult {
		fmt.Fprintf(&b, "video %d: %s\n", i+1, v.URL)
	}
	return b.String(), nil
}

func transcribe(ctx context.Context, c *client.Client, path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	resp, err := c.Audio().Transcribe(ctx, client.AudioTranscriptionRequest{FileName: filepath.Base(path), FileData: data})
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

func parseLayout(ctx context.Context, c *client.Client, target string) (string, error) {
	file, err := fileinput.FileOrURL(target)
	if err != nil {
		return "", err
	}
	resp, err := c.Layout().Parse(ctx, client.LayoutParsingRequest{File: file})
	if err != nil {
		return "", err
	}
	return resp.MDResults, nil
}
