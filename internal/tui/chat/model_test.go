package chat

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

func noClient() *client.Client { return nil }

// Streamed content and reasoning accumulate; a clean end commits both to the
// transcript (the reasoning is echoed back on the next turn).
func TestChunksAccumulateAndCommit(t *testing.T) {
	m := New(noClient, 0)
	m.stream = &stream{next: func() (client.StreamChunk, error, bool) { return client.StreamChunk{}, nil, false }, stop: func() {}, cancel: func() {}}
	if !m.Streaming() {
		t.Fatal("expected Streaming() with a stream in flight")
	}
	var next tea.Model = m
	for _, d := range []client.StreamDelta{{ReasoningContent: "think"}, {Content: "Hel"}, {Content: "lo"}} {
		next, _ = next.(Model).Update(chunkMsg{Choices: []client.StreamChoice{{Delta: d}}})
	}
	done, cmd := next.(Model).Update(streamDoneMsg{})
	got := done.(Model)
	if got.Streaming() || cmd != nil {
		t.Fatal("a clean end clears the stream without an error")
	}
	last := got.messages[len(got.messages)-1]
	if last.Content != "Hello" || last.ReasoningContent != "think" {
		t.Errorf("committed %+v", last)
	}
}

// A user cancel ends the stream without an error toast; ctrl+c cancels
// without arming another pull (the pending pull delivers the end).
func TestCtrlCCancels(t *testing.T) {
	m := New(noClient, 0)
	cancelled := false
	m.stream = &stream{cancel: func() { cancelled = true }}
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if !cancelled || cmd != nil {
		t.Fatalf("ctrl+c: cancelled=%v cmd=%v; want cancel and no new pull", cancelled, cmd)
	}
	if _, cmd := m.Update(streamDoneMsg{err: context.Canceled}); cmd != nil {
		t.Error("a cancelled stream must not raise an error")
	}
}

// End to end: each wait pulls one chunk, routed to the chat tab, until the
// stream ends.
func TestStreamPullsRoutedChunks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, c := range []string{"a", "b"} {
			fmt.Fprintf(w, "data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":%q}}]}\n\n", c)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	c, err := client.NewClient(client.Config{APIKey: "k", BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	s, cmd := startStream(c, client.ChatRequest{Model: "m", Messages: []client.Message{{Role: "user", Content: "hi"}}}, 7)
	var got string
	for {
		routed, ok := cmd().(uimsg.Routed)
		if !ok || routed.Tab != 7 {
			t.Fatalf("expected a message routed to tab 7, got %#v", routed)
		}
		if done, ok := routed.Msg.(streamDoneMsg); ok {
			if done.err != nil {
				t.Fatalf("stream error: %v", done.err)
			}
			break
		}
		got += routed.Msg.(chunkMsg).Choices[0].Delta.Content
		cmd = s.wait()
	}
	if got != "ab" {
		t.Errorf("streamed %q, want ab", got)
	}
}

// ctrl+o asks the root for the model picker, and SetModel switches models.
func TestModelPicker(t *testing.T) {
	m := New(noClient, 0)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})
	if _, ok := cmd().(uimsg.OpenModelPicker); !ok {
		t.Fatal("ctrl+o should open the model picker")
	}
	next, _ := m.SetModel(client.DefaultFastModel)
	if next.(Model).ModelID() != client.DefaultFastModel {
		t.Error("SetModel did not switch")
	}
}
