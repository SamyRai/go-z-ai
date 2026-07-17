package chat

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// Streaming reflects the in-flight flag the root model reads to block nav.
func TestStreamingFlag(t *testing.T) {
	m := New(nil)
	if m.Streaming() {
		t.Error("a fresh chat model is not streaming")
	}
	m.streaming = true
	if !m.Streaming() {
		t.Error("expected Streaming() true while streaming")
	}
}

// Streamed chunks accumulate into the pending assistant reply; a clean
// streamDoneMsg commits it to the transcript and stops streaming.
func TestChunkAccumulationAndDone(t *testing.T) {
	m := New(nil)
	m.streaming = true

	next, _ := m.Update(chunkMsg{Choices: []client.StreamChoice{{Delta: client.StreamDelta{Content: "Hel"}}}})
	next, _ = next.(Model).Update(chunkMsg{Choices: []client.StreamChoice{{Delta: client.StreamDelta{Content: "lo"}}}})
	if got := next.(Model).pending; got != "Hello" {
		t.Errorf("expected pending 'Hello', got %q", got)
	}

	done, _ := next.(Model).Update(streamDoneMsg{})
	got := done.(Model)
	if got.streaming {
		t.Error("expected streaming cleared on done")
	}
	if got.pending != "" {
		t.Error("expected pending flushed on done")
	}
	if len(got.messages) == 0 || got.messages[len(got.messages)-1].Content != "Hello" {
		t.Errorf("expected the assistant reply committed to messages, got %+v", got.messages)
	}
}

// A user-initiated cancel (context.Canceled) ends the stream without raising an
// error toast.
func TestStreamCancelIsNotAnError(t *testing.T) {
	m := New(nil)
	m.streaming = true

	_, cmd := m.Update(streamDoneMsg{err: context.Canceled})
	if cmd != nil {
		t.Error("expected no error toast on a cancelled stream")
	}
}

// ctrl+c while streaming cancels the in-flight stream instead of quitting.
func TestCtrlCCancelsStream(t *testing.T) {
	m := New(nil)
	cancelled := false
	m.streaming = true
	m.handle = streamHandle{cancel: func() { cancelled = true }}

	m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl, Text: ""})
	if !cancelled {
		t.Error("expected ctrl+c to call the stream's cancel func")
	}
}
