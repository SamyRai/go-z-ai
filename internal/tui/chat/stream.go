package chat

import (
	"context"
	"iter"

	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

// chunkMsg carries one streamed delta.
type chunkMsg client.StreamChunk

// streamDoneMsg ends a stream: err is nil on a clean finish and
// context.Canceled when the user aborted it.
type streamDoneMsg struct{ err error }

// stream adapts ChatService.Stream's iterator to Bubble Tea's message loop:
// each wait command pulls exactly one value, and the next wait is issued only
// after its message is handled, so pulls never overlap and there is no
// goroutine or channel to leak.
type stream struct {
	next   func() (client.StreamChunk, error, bool)
	stop   func()
	cancel context.CancelFunc
	tab    int // results are routed to the chat tab, whichever tab is active
}

func startStream(c *client.Client, req client.ChatRequest, tab int) (*stream, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())
	next, stop := iter.Pull2(c.Chat().Stream(ctx, req))
	s := &stream{next: next, stop: stop, cancel: cancel, tab: tab}
	return s, s.wait()
}

// wait pulls the next chunk (or the end of the stream).
func (s *stream) wait() tea.Cmd {
	return uimsg.Route(s.tab, func() tea.Msg {
		chunk, err, ok := s.next()
		switch {
		case !ok:
			s.release()
			return streamDoneMsg{}
		case err != nil:
			s.release()
			return streamDoneMsg{err: err}
		}
		return chunkMsg(chunk)
	})
}

// release frees the iterator and the request context.
func (s *stream) release() {
	s.stop()
	s.cancel()
}
