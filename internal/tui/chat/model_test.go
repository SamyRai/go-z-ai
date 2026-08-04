package chat

import (
	"context"
	"errors"
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
// It must also re-arm the chunk pump so the producer's channel-close drains
// into a streamDoneMsg — otherwise m.streaming stays true forever and the tab
// is wedged (can't send again, can't tab away).
func TestCtrlCCancelsStream(t *testing.T) {
	m := New(nil)
	cancelled := false
	m.streaming = true
	// Simulate the post-cancel pump state: the producer goroutine has broken
	// out of its range on ctx cancel, sent ctx.Err() on done, and closed ch.
	ch := make(chan client.StreamChunk)
	close(ch)
	done := make(chan error, 1)
	done <- context.Canceled
	m.handle = streamHandle{
		ch:     ch,
		done:   done,
		cancel: func() { cancelled = true },
	}

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl, Text: ""})
	if !cancelled {
		t.Fatal("expected ctrl+c to call the stream's cancel func")
	}
	if cmd == nil {
		t.Fatal("expected ctrl+c to re-arm the chunk pump (non-nil cmd); nil cmd wedges m.streaming=true forever")
	}
	// Executing the re-armed cmd must drain the closed channel into a
	// streamDoneMsg carrying context.Canceled — the only path that clears
	// m.streaming.
	msg := cmd()
	ds, ok := msg.(streamDoneMsg)
	if !ok {
		t.Fatalf("expected streamDoneMsg from re-armed pump, got %T", msg)
	}
	if !errors.Is(ds.err, context.Canceled) {
		t.Fatalf("expected streamDoneMsg.err = context.Canceled, got %v", ds.err)
	}
}
