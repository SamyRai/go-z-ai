package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestStreamContent verifies the iterator delivers each SSE delta in order
// and terminates cleanly on [DONE]. Mirrors TestCreateStreamContent.
func TestStreamContent(t *testing.T) {
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
	for chunk, err := range c.Chat().Stream(context.Background(), req) {
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		chunks++
		if len(chunk.Choices) > 0 {
			got.WriteString(chunk.Choices[0].Delta.Content)
		}
	}
	if got.String() != "Hello" {
		t.Errorf("expected content 'Hello', got %q", got.String())
	}
	if chunks != 3 {
		t.Errorf("expected 3 chunks, got %d", chunks)
	}
}

// TestStreamRetriableThenStream verifies connect-phase retry: a 429 on the
// first attempt must be retried, then the stream proceeds. Mirrors
// TestCreateStreamRetriableThenStream.
func TestStreamRetriableThenStream(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"1302","message":"Rate limit reached"}}`)
			return
		}
		h := sseHandler(
			`{"id":"1","model":"m","choices":[{"index":0,"delta":{"content":"ok"}}]}`,
			`[DONE]`,
		)
		h.ServeHTTP(w, r)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 3})
	req := ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}, TopP: 0.95}

	var got strings.Builder
	for chunk, err := range c.Chat().Stream(context.Background(), req) {
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(chunk.Choices) > 0 {
			got.WriteString(chunk.Choices[0].Delta.Content)
		}
	}
	if got.String() != "ok" {
		t.Errorf("expected 'ok', got %q", got.String())
	}
	if calls != 2 {
		t.Errorf("expected 2 calls (retry + success), got %d", calls)
	}
}

// TestStreamNonRetriable verifies a non-retriable error surfaces immediately
// without retry. The iterator yields the error as the terminal item.
// Uses code 1308 (usage limit, non-retriable) matching TestCreateStreamNonRetriable.
func TestStreamNonRetriable(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"1308","message":"usage limit"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 3})
	req := ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}, TopP: 0.95}

	chunks, errs := 0, 0
	var firstErr error
	for chunk, err := range c.Chat().Stream(context.Background(), req) {
		if err != nil {
			errs++
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		chunks++
		_ = chunk
	}
	if chunks != 0 {
		t.Errorf("expected 0 chunks, got %d", chunks)
	}
	if errs != 1 {
		t.Errorf("expected 1 error, got %d", errs)
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry), got %d", calls)
	}
	var apiErr *APIError
	if !errors.As(firstErr, &apiErr) {
		t.Errorf("expected *APIError, got %T: %v", firstErr, firstErr)
	}
}

// TestStreamValidationErrPreConnect verifies a request-validation failure
// surfaces as the iterator's terminal err (not a panic, not a separate
// return value).
func TestStreamValidationErrPreConnect(t *testing.T) {
	c := newTestClient(t, "http://unreachable.invalid", Config{})
	// Missing model + messages — fails validateChatRequest before any HTTP.
	req := ChatRequest{}

	errs := 0
	for _, err := range c.Chat().Stream(context.Background(), req) {
		if err != nil {
			errs++
			if !strings.Contains(err.Error(), "invalid chat request") {
				t.Errorf("expected validation err, got %v", err)
			}
		}
	}
	if errs != 1 {
		t.Errorf("expected exactly 1 validation err, got %d", errs)
	}
}

// TestStreamContextCancel verifies cancelling the context tears down the
// in-flight stream without leaking the producer goroutine. This is the
// critical correctness property of the iterator design.
func TestStreamContextCancel(t *testing.T) {
	// Server that streams slowly: one chunk, then blocks forever.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprintf(w, "data: {\"id\":\"1\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"x\"}}]}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		// Block until the client disconnects (ctx cancelled) — simulates a
		// long-running stream the caller wants to abort.
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	req := ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}, TopP: 0.95}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after the first chunk arrives.
	gotFirst := make(chan struct{})
	go func() {
		<-gotFirst
		cancel()
	}()

	startGoroutines := runtime.NumGoroutine()
	chunks := 0
	for chunk, err := range c.Chat().Stream(ctx, req) {
		if err != nil {
			// ctx cancellation surfaces as an err — acceptable.
			break
		}
		chunks++
		if chunks == 1 {
			close(gotFirst)
		}
		_ = chunk
	}
	cancel() // idempotent

	// Verify the producer goroutine exited (no leak). Allow a brief grace.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= startGoroutines {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := runtime.NumGoroutine(); got > startGoroutines {
		t.Errorf("goroutine leak: started with %d, now %d", startGoroutines, got)
	}
}

// TestStreamEarlyBreakNoCancel is the regression test for a deadlock in
// pullStream: a caller that `break`s out of the range WITHOUT cancelling the
// context must not hang. Before the fix, the consumer waited on producerDone
// while the producer blocked on its channel send (the only unblock was
// ctx.Done(), which never fired) — a deterministic hang + goroutine/body
// leak. Now pullStream owns an internal context it cancels on consumer exit,
// unblocking the producer. The load-bearing assertion is that the range loop
// returns promptly after break (a hang is the failure mode); the goroutine
// delta is a secondary check (relaxed, since httptest's server keeps a
// transient background-read goroutine that muddies an exact count).
func TestStreamEarlyBreakNoCancel(t *testing.T) {
	// handlerDone lets the handler exit promptly when the test tears down, so
	// srv.Close() never blocks on a handler stuck in <-r.Context().Done().
	// handlerStop lets the test unblock the handler so srv.Close() returns
	// promptly; without it the handler (stuck in <-r.Context().Done()) would
	// make defer srv.Close() hang.
	handlerStop := make(chan struct{})
	handlerExited := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(handlerExited)
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprintf(w, "data: {\"id\":\"1\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"x\"}}]}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		select {
		case <-r.Context().Done(): // client disconnected
		case <-handlerStop: // test tearing down
		}
	}))
	defer func() {
		close(handlerStop) // unblock the handler
		<-handlerExited    // wait for it to return
		srv.Close()
	}()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	req := ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}, TopP: 0.95}

	startGoroutines := runtime.NumGoroutine()
	chunks := 0
	returned := make(chan struct{})
	go func() {
		// Deliberately DO NOT cancel ctx — this is the bug's trigger.
		for chunk, err := range c.Chat().Stream(context.Background(), req) {
			if err != nil {
				t.Errorf("unexpected err on first chunk: %v", err)
				return
			}
			chunks++
			_ = chunk
			break // early exit, no cancel — the scenario that used to deadlock
		}
		close(returned)
	}()

	// The core regression assertion: the loop must return, not hang.
	select {
	case <-returned:
	case <-time.After(3 * time.Second):
		t.Fatal("range loop hung after early break — B1 deadlock present")
	}

	// Secondary: the client's producer goroutine should have exited. Allow a
	// brief grace; tolerate the httptest server's transient background-read
	// goroutine by checking the delta settles to at most +1 (server noise).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= startGoroutines+1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := runtime.NumGoroutine(); got > startGoroutines+1 {
		t.Errorf("client goroutine leak after early break: started %d, now %d (tolerance +1)", startGoroutines, got)
	}
	if chunks != 1 {
		t.Errorf("expected 1 chunk before break, got %d", chunks)
	}
}

// TestStreamCallbackDelegation confirms the deprecated CreateStream still
// works by delegating to Stream. This is the equivalence guarantee for
// existing callers until v1.0 removal.
func TestStreamCallbackDelegation(t *testing.T) {
	srv := httptest.NewServer(sseHandler(
		`{"id":"1","model":"m","choices":[{"index":0,"delta":{"content":"delegated"}}]}`,
		`[DONE]`,
	))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
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
	if got.String() != "delegated" {
		t.Errorf("expected 'delegated', got %q", got.String())
	}
}

// TestAnthropicStreamContent verifies the Anthropic iterator path mirrors the
// chat iterator's basic content-delivery semantics, using Anthropic's
// event:/data: SSE framing. Uses newRedirectingTestClient because the
// Anthropic endpoint hardcodes AnthropicBaseURL (not Config.BaseURL).
func TestAnthropicStreamContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprint(w, "event: message_start\n")
		fmt.Fprint(w, `data: {"type":"message_start","message":{"id":"msg_1"}}`+"\n\n")
		fmt.Fprint(w, "event: content_block_delta\n")
		fmt.Fprint(w, `data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"Hi"}}`+"\n\n")
		fmt.Fprint(w, "event: message_stop\n")
		fmt.Fprint(w, `data: {"type":"message_stop"}`+"\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer srv.Close()

	c := newRedirectingTestClient(t, srv, Config{MaxRetries: 0})
	req := AnthropicMessageRequest{
		Model:     "glm-4.6",
		MaxTokens: 100,
		Messages:  []AnthropicMessage{AnthropicTextMessage("user", "hi")},
	}

	var events int
	var lastType string
	for ev, err := range c.Anthropic().Stream(context.Background(), req) {
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		events++
		lastType = ev.Type
	}
	if events != 3 {
		t.Errorf("expected 3 events, got %d", events)
	}
	if lastType != "message_stop" {
		t.Errorf("expected last event type 'message_stop', got %q", lastType)
	}
}

// spanPropHook is a Hook that, on OnRequest, stamps a sentinel value into the
// returned context. If the stream iterator correctly propagates the connect
// attempt's hookCtx to OnStreamChunk / OnResponse, those hooks observe the
// sentinel; if it does not (the pre-fix bug, where OnStreamChunk received the
// span-less streamCtx), they observe nil. This is the in-package proxy for
// "the context contains the span" — the OTel hook stashes its span under a
// private key the same way.
type spanPropHook struct {
	markKey any
	// recorded marks whether OnStreamChunk/OnResponse saw the hook-attached ctx.
	chunkCtxSeen  any
	respCtxSeen   any
	chunkMetas    []RequestMeta
	respMeta      *ResponseMeta
	chunkCount    int
	respCount     int
	onRequestReqs []RequestMeta
}

func (h *spanPropHook) OnRequest(ctx context.Context, meta RequestMeta) context.Context {
	h.onRequestReqs = append(h.onRequestReqs, meta)
	return context.WithValue(ctx, h.markKey, "hook-attached")
}
func (h *spanPropHook) OnResponse(ctx context.Context, meta ResponseMeta) {
	h.respCtxSeen = ctx.Value(h.markKey)
	h.respMeta = &meta
	h.respCount++
}
func (h *spanPropHook) OnError(context.Context, RequestMeta, error) {}
func (h *spanPropHook) OnStreamChunk(ctx context.Context, meta RequestMeta, _ any) {
	h.chunkCtxSeen = ctx.Value(h.markKey)
	h.chunkMetas = append(h.chunkMetas, meta)
	h.chunkCount++
}

// TestStreamSpanLifecycle is the regression test for the stream span-lifecycle
// bug. Before the fix:
//
//  1. connectChatStream built hookCtx (with the OTel span under a private key)
//     inside the retry loop but never returned it, so the Stream wrapper
//     invoked OnStreamChunk with the span-less streamCtx — chunk hooks saw no
//     span and recorded nothing.
//  2. The stream path never called OnResponse on clean completion, so the
//     per-attempt span was never ended → leaked to the exporter.
//  3. The chunk/error meta's Attempt was hardcoded to 0 even when a retry
//     succeeded.
//
// This test asserts all three are fixed on a clean-stream-completion path that
// succeeds on the second connect attempt: OnResponse fires once (span ended),
// OnStreamChunk receives a context carrying the hook-attached sentinel (span
// present), and the chunk/response meta carries Attempt=1 (the successful
// retry index).
func TestStreamSpanLifecycle(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			// Retriable 429 on attempt 0 → connect retries.
			writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"1302","message":"rate limit"}}`)
			return
		}
		// Attempt 1 succeeds: stream two chunks then [DONE].
		sseHandler(
			`{"id":"1","model":"m","choices":[{"index":0,"delta":{"content":"a"}}]}`,
			`{"id":"1","model":"m","choices":[{"index":0,"delta":{"content":"b"}}]}`,
			`[DONE]`,
		).ServeHTTP(w, r)
	}))
	defer srv.Close()

	type key int
	const mark key = 1
	h := &spanPropHook{markKey: mark}
	c := newTestClient(t, srv.URL, Config{MaxRetries: 3, Hooks: []Hook{h}})
	req := ChatRequest{Model: "glm-test", Messages: []Message{{Role: "user", Content: "hi"}}, TopP: 0.95}

	chunks := 0
	for chunk, err := range c.Chat().Stream(context.Background(), req) {
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		_ = chunk
		chunks++
	}
	if chunks != 2 {
		t.Fatalf("expected 2 chunks, got %d", chunks)
	}
	if calls != 2 {
		t.Fatalf("expected 2 connect calls (retry then success), got %d", calls)
	}

	// (a) OnResponse fired exactly once on clean completion (span ended).
	if h.respCount != 1 {
		t.Errorf("expected exactly 1 OnResponse on clean stream completion, got %d", h.respCount)
	}
	// (b) OnStreamChunk + OnResponse received a context carrying the
	// hook-attached sentinel (i.e. the connect attempt's hookCtx propagated,
	// not the span-less streamCtx). This is the core regression assertion.
	if h.chunkCtxSeen != "hook-attached" {
		t.Errorf("OnStreamChunk did not receive the hook-attached ctx (span missing): got %v", h.chunkCtxSeen)
	}
	if h.respCtxSeen != "hook-attached" {
		t.Errorf("OnResponse did not receive the hook-attached ctx (span missing): got %v", h.respCtxSeen)
	}
	// (c) Chunk + response meta carry the SUCCESSFUL connect attempt index (1),
	// not the hardcoded 0. OnRequest fired twice (attempts 0 and 1).
	if len(h.onRequestReqs) != 2 {
		t.Errorf("expected 2 OnRequest calls (one per attempt), got %d", len(h.onRequestReqs))
	}
	if h.chunkCount != 2 {
		t.Errorf("expected 2 OnStreamChunk calls, got %d", h.chunkCount)
	}
	for i, m := range h.chunkMetas {
		if m.Attempt != 1 {
			t.Errorf("chunk[%d].Attempt = %d, want 1 (successful retry index)", i, m.Attempt)
		}
	}
	if h.respMeta == nil {
		t.Fatal("expected non-nil ResponseMeta on clean completion")
	} else if h.respMeta.Attempt != 1 {
		t.Errorf("response Attempt = %d, want 1", h.respMeta.Attempt)
	} else if h.respMeta.StatusCode != http.StatusOK {
		t.Errorf("response StatusCode = %d, want 200", h.respMeta.StatusCode)
	}
}

// TestStreamSpanLifecycleMidStreamError is the error-path companion: a stream
// that connects cleanly then fails mid-read must fire OnError (not OnResponse)
// against the captured attempt context, so the span ends with ERROR status.
func TestStreamSpanLifecycleMidStreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Connect succeeds, then the body delivers malformed SSE so readSSE
		// returns a parse error mid-stream.
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprint(w, "data: {\"id\":\"1\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"a\"}}]}\n\n")
		fmt.Fprint(w, "data: {not valid json\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer srv.Close()

	type key int
	const mark key = 1
	h := &spanPropHook{markKey: mark}
	c := newTestClient(t, srv.URL, Config{MaxRetries: 0, Hooks: []Hook{h}})
	req := ChatRequest{Model: "glm-test", Messages: []Message{{Role: "user", Content: "hi"}}, TopP: 0.95}

	var sawErr error
	for _, err := range c.Chat().Stream(context.Background(), req) {
		if err != nil {
			sawErr = err
		}
	}
	if sawErr == nil {
		t.Fatal("expected a mid-stream parse error")
	}
	// OnResponse must NOT fire (the stream errored, not completed cleanly).
	if h.respCount != 0 {
		t.Errorf("expected 0 OnResponse on mid-stream error, got %d", h.respCount)
	}
	// The one chunk delivered before the error still saw the hook-attached ctx.
	if h.chunkCtxSeen != "hook-attached" {
		t.Errorf("OnStreamChunk did not receive hook-attached ctx: got %v", h.chunkCtxSeen)
	}
}
