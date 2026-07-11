package chat

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"zai-api-client/pkg/client"
)

// chunkMsg carries one streamed delta.
type chunkMsg client.StreamChunk

// streamDoneMsg signals the stream ended (err is nil on a clean finish, or
// context.Canceled when the user aborted mid-stream via ctrl+c).
type streamDoneMsg struct{ err error }

// streamHandle bridges ChatService.CreateStream's blocking onChunk callback
// (run on a goroutine) into Bubble Tea's message loop via channels — the
// standard Bubble Tea idiom for wrapping an external streaming source.
type streamHandle struct {
	ch     chan client.StreamChunk
	done   chan error
	cancel context.CancelFunc
}

// startStream launches req on a goroutine and returns the tea.Cmd that
// begins draining it, plus the handle needed to cancel it mid-stream.
func startStream(c *client.Client, req client.ChatRequest) (tea.Cmd, streamHandle) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan client.StreamChunk, 16)
	done := make(chan error, 1)
	h := streamHandle{ch: ch, done: done, cancel: cancel}

	go func() {
		err := c.Chat().CreateStream(ctx, req, func(chunk client.StreamChunk) error {
			select {
			case ch <- chunk:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		done <- err
		close(ch)
	}()

	return waitForChunk(h), h
}

// waitForChunk drains one event off the stream and must be re-issued after
// every chunkMsg until a streamDoneMsg arrives.
func waitForChunk(h streamHandle) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-h.ch
		if !ok {
			return streamDoneMsg{err: <-h.done}
		}
		return chunkMsg(chunk)
	}
}
