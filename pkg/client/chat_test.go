package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// sseHandler writes the given SSE frames (each becomes a `data:` line), with a
// final `data: [DONE]`, flushing between frames so it behaves like a real stream.
func sseHandler(frames ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for _, f := range frames {
			fmt.Fprintf(w, "data: %s\n\n", f)
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}

// A normal content stream should deliver each delta in order and terminate on
// [DONE]; accumulated content must reconstruct the full message.
func TestCreateStreamContent(t *testing.T) {
	srv := httptest.NewServer(sseHandler(
		`{"id":"1","model":"m","choices":[{"index":0,"delta":{"role":"assistant","content":"Hel"}}]}`,
		`{"id":"1","model":"m","choices":[{"index":0,"delta":{"content":"lo"}}]}`,
		`{"id":"1","model":"m","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`[DONE]`,
	))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	req := ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}, TopP: 0.95}

	var got strings.Builder
	var chunks int
	err := c.Chat().CreateStream(context.Background(), req, func(ch StreamChunk) error {
		chunks++
		if len(ch.Choices) > 0 {
			got.WriteString(ch.Choices[0].Delta.Content)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("CreateStream: %v", err)
	}
	if got.String() != "Hello" {
		t.Fatalf("expected content 'Hello', got %q", got.String())
	}
	if chunks != 3 {
		t.Fatalf("expected 3 chunks, got %d", chunks)
	}
}

// SSE comments (`:`), keep-alives, and non-data control lines must be ignored.
func TestCreateStreamIgnoresControlLines(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, ": keep-alive\n\n")
		fmt.Fprint(w, "event: ping\n\n")
		fmt.Fprint(w, "data: {\"id\":\"1\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	req := ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}, TopP: 0.95}

	var got strings.Builder
	err := c.Chat().CreateStream(context.Background(), req, func(ch StreamChunk) error {
		if len(ch.Choices) > 0 {
			got.WriteString(ch.Choices[0].Delta.Content)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("CreateStream: %v", err)
	}
	if got.String() != "ok" {
		t.Fatalf("expected single 'ok' delta, got %q", got.String())
	}
}

// A retriable failure before the stream starts is retried, then streamed.
func TestCreateStreamRetriableThenStream(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"1302","message":"rate limit"}}`)
			return
		}
		sseHandler(`{"id":"1","model":"m","choices":[{"index":0,"delta":{"content":"Hi"}}]}`, `[DONE]`)(w, r)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 2})
	req := ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}, TopP: 0.95}

	var got strings.Builder
	if err := c.Chat().CreateStream(context.Background(), req, func(ch StreamChunk) error {
		if len(ch.Choices) > 0 {
			got.WriteString(ch.Choices[0].Delta.Content)
		}
		return nil
	}); err != nil {
		t.Fatalf("CreateStream: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls (1 retry), got %d", calls)
	}
	if got.String() != "Hi" {
		t.Fatalf("expected 'Hi', got %q", got.String())
	}
}

// An onChunk error aborts the stream and is returned to the caller.
func TestCreateStreamAbortOnError(t *testing.T) {
	srv := httptest.NewServer(sseHandler(
		`{"id":"1","model":"m","choices":[{"index":0,"delta":{"content":"a"}}]}`,
		`{"id":"1","model":"m","choices":[{"index":0,"delta":{"content":"b"}}]}`,
		`[DONE]`,
	))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	req := ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}, TopP: 0.95}

	stop := fmt.Errorf("stop requested")
	var count int
	err := c.Chat().CreateStream(context.Background(), req, func(ch StreamChunk) error {
		count++
		if count == 1 {
			return stop
		}
		return nil
	})
	if err != stop {
		t.Fatalf("expected abort error, got %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 chunk before abort, got %d", count)
	}
}

// A non-retriable error (quota 1308) is not retried and is returned as APIError.
func TestCreateStreamNonRetriable(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"1308","message":"usage limit"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 3})
	req := ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}, TopP: 0.95}
	err := c.Chat().CreateStream(context.Background(), req, func(ch StreamChunk) error { return nil })
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call (no retry), got %d", calls)
	}
	if _, ok := err.(*APIError); !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
}

// A stream that runs longer than Config.Timeout must not be cut off:
// Timeout bounds connection setup/response headers, not body reads, so a
// long-lived SSE stream survives past it as long as tokens keep arriving.
func TestCreateStreamSurvivesPastConfigTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for i := 0; i < 3; i++ {
			fmt.Fprint(w, "data: {\"id\":\"1\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"x\"}}]}\n\n")
			if flusher != nil {
				flusher.Flush()
			}
			time.Sleep(30 * time.Millisecond)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	// A 10ms Config.Timeout would kill the stream almost immediately under
	// the old http.Client.Timeout-based design; the transport-level timeout
	// only bounds dial/handshake/header-wait, so the ~90ms stream (3x30ms)
	// must still complete cleanly.
	c := newTestClient(t, srv.URL, Config{MaxRetries: 0, Timeout: 10 * time.Millisecond})
	req := ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}, TopP: 0.95}

	var chunks int
	err := c.Chat().CreateStream(context.Background(), req, func(ch StreamChunk) error {
		chunks++
		return nil
	})
	if err != nil {
		t.Fatalf("CreateStream: %v", err)
	}
	if chunks != 3 {
		t.Fatalf("expected 3 chunks despite a short Config.Timeout, got %d", chunks)
	}
}

// Context cancellation aborts an in-flight stream.
func TestCreateStreamContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for i := 0; i < 3; i++ {
			fmt.Fprintf(w, "data: {\"id\":\"1\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"x\"}}]}\n\n")
			if flusher != nil {
				flusher.Flush()
			}
			time.Sleep(20 * time.Millisecond)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	req := ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}, TopP: 0.95}
	err := c.Chat().CreateStream(ctx, req, func(ch StreamChunk) error { return nil })
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}

// CreateAsync posts to /async/chat/completions (not /chat/completions) and
// returns the task immediately, without a completion body.
func TestChatCreateAsync(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeJSON(w, http.StatusOK, `{"model":"glm-4.6","id":"task-1","request_id":"req-1","task_status":"PROCESSING"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	resp, err := c.Chat().CreateAsync(context.Background(), ChatRequest{
		Model:    "glm-4.6",
		Messages: []Message{{Role: "user", Content: "hi"}},
		TopP:     0.95,
	})
	if err != nil {
		t.Fatalf("CreateAsync: %v", err)
	}
	if gotPath != "/async/chat/completions" {
		t.Errorf("expected path /async/chat/completions, got %q", gotPath)
	}
	if resp.ID != "task-1" || resp.TaskStatus != TaskStatusProcessing {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// CreateAsync rejects Stream: true — streaming isn't meaningful for a
// fire-and-forget async submission.
func TestChatCreateAsyncRejectsStream(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	_, err := c.Chat().CreateAsync(context.Background(), ChatRequest{
		Model:    "glm-4.6",
		Messages: []Message{{Role: "user", Content: "hi"}},
		TopP:     0.95,
		Stream:   true,
	})
	if err == nil {
		t.Error("expected error when Stream is true")
	}
}

// GetAsyncResult parses Choices/Usage when polling a completed async chat
// completion task (not Data/VideoResult, which are for image/video tasks).
func TestGetAsyncResultParsesChatCompletion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"id":"task-1","model":"glm-4.6","task_status":"SUCCESS","choices":[{"index":0,"message":{"role":"assistant","content":"hi there"}}],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	resp, err := c.GetAsyncResult(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("GetAsyncResult: %v", err)
	}
	if len(resp.Choices) != 1 || resp.Choices[0].Message.Content != "hi there" {
		t.Fatalf("unexpected choices: %+v", resp.Choices)
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 5 {
		t.Fatalf("unexpected usage: %+v", resp.Usage)
	}
}
