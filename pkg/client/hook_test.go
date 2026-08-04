package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// recordingHook captures every hook invocation for assertion. Safe for
// concurrent use (OnStreamChunk fires from the SSE reader goroutine).
type recordingHook struct {
	mu        sync.Mutex
	requests  []RequestMeta
	responses []ResponseMeta
	errors    []hookErr
	chunks    []RequestMeta
}

type hookErr struct {
	meta RequestMeta
	err  error
}

func (h *recordingHook) OnRequest(_ context.Context, meta RequestMeta) context.Context {
	h.mu.Lock()
	h.requests = append(h.requests, meta)
	h.mu.Unlock()
	return context.Background()
}
func (h *recordingHook) OnResponse(_ context.Context, meta ResponseMeta) {
	h.mu.Lock()
	h.responses = append(h.responses, meta)
	h.mu.Unlock()
}
func (h *recordingHook) OnError(_ context.Context, meta RequestMeta, err error) {
	h.mu.Lock()
	h.errors = append(h.errors, hookErr{meta, err})
	h.mu.Unlock()
}
func (h *recordingHook) OnStreamChunk(_ context.Context, meta RequestMeta, _ any) {
	h.mu.Lock()
	h.chunks = append(h.chunks, meta)
	h.mu.Unlock()
}

// TestHookNoConfigZeroCost verifies a client with no hooks configured does
// not invoke any hook methods (nil-safe fast path).
func TestHookNoConfigZeroCost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"id":"x","model":"m","choices":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{}) // no Hooks
	if len(c.hooks) != 0 {
		t.Fatalf("expected 0 hooks, got %d", len(c.hooks))
	}
	var resp ChatResponse
	if err := c.doRequest(context.Background(), "POST", "/chat/completions", map[string]string{"q": "hi"}, &resp); err != nil {
		t.Fatalf("doRequest: %v", err)
	}
	// No panic, no surprise — the no-hook path must be transparent.
}

// TestHookFiresOnResponse verifies a successful non-streaming request fires
// OnRequest once and OnResponse once, with no OnError. Also confirms the
// Usage is extracted from the typed response.
func TestHookFiresOnResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"id":"x","model":"m","choices":[],"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}`)
	}))
	defer srv.Close()

	h := &recordingHook{}
	c := newTestClient(t, srv.URL, Config{Hooks: []Hook{h}})

	var resp ChatResponse
	if err := c.doRequest(context.Background(), "POST", "/chat/completions", map[string]string{"q": "hi"}, &resp); err != nil {
		t.Fatalf("doRequest: %v", err)
	}
	if got := len(h.requests); got != 1 {
		t.Errorf("expected 1 OnRequest call, got %d", got)
	}
	if got := len(h.responses); got != 1 {
		t.Fatalf("expected 1 OnResponse call, got %d", got)
	}
	if h.responses[0].StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", h.responses[0].StatusCode)
	}
	if h.responses[0].Usage == nil {
		t.Fatal("expected non-nil Usage on response meta")
	}
	if h.responses[0].Usage.TotalTokens != 8 {
		t.Errorf("expected 8 total tokens, got %d", h.responses[0].Usage.TotalTokens)
	}
	if len(h.errors) != 0 {
		t.Errorf("expected 0 OnError calls, got %d", len(h.errors))
	}
}

// TestHookFiresOnError verifies a non-retriable failure fires OnRequest and
// OnError but not OnResponse.
func TestHookFiresOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"1308","message":"usage limit"}}`)
	}))
	defer srv.Close()

	h := &recordingHook{}
	c := newTestClient(t, srv.URL, Config{MaxRetries: 0, Hooks: []Hook{h}})

	var resp ChatResponse
	err := c.doRequest(context.Background(), "POST", "/chat/completions", map[string]string{"q": "hi"}, &resp)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := len(h.requests); got != 1 {
		t.Errorf("expected 1 OnRequest call, got %d", got)
	}
	if got := len(h.errors); got != 1 {
		t.Fatalf("expected 1 OnError call, got %d", got)
	}
	if len(h.responses) != 0 {
		t.Errorf("expected 0 OnResponse calls, got %d", len(h.responses))
	}
	var apiErr *APIError
	if !errors.As(h.errors[0].err, &apiErr) {
		t.Errorf("expected *APIError in hook, got %T", h.errors[0].err)
	}
}

// TestHookFiresOnRetry verifies a retried-then-successful request fires
// OnRequest twice (one per attempt) and OnResponse once (only on success).
func TestHookFiresOnRetry(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"1302","message":"rate limit"}}`)
			return
		}
		writeJSON(w, http.StatusOK, `{"id":"x","model":"m","choices":[]}`)
	}))
	defer srv.Close()

	h := &recordingHook{}
	c := newTestClient(t, srv.URL, Config{MaxRetries: 3, Hooks: []Hook{h}})

	var resp ChatResponse
	if err := c.doRequest(context.Background(), "POST", "/chat/completions", map[string]string{"q": "hi"}, &resp); err != nil {
		t.Fatalf("doRequest: %v", err)
	}
	if got := len(h.requests); got != 2 {
		t.Errorf("expected 2 OnRequest calls (1 per attempt), got %d", got)
	}
	if h.requests[0].Attempt != 0 || h.requests[1].Attempt != 1 {
		t.Errorf("expected attempts 0 then 1, got %d and %d", h.requests[0].Attempt, h.requests[1].Attempt)
	}
	if got := len(h.responses); got != 1 {
		t.Errorf("expected 1 OnResponse (only on success), got %d", got)
	}
	if h.responses[0].Attempt != 1 {
		t.Errorf("expected OnResponse on attempt 1, got %d", h.responses[0].Attempt)
	}
}

// TestRetryAttemptEndsSpan is the regression test for the retry-path span leak.
// Before the fix: a failed attempt that was retried skipped OnError, so the
// OTel span started by OnRequest for that attempt was never ended (leaked to
// the exporter). With the fix, OnError fires once for the retried attempt 0
// (the 429) AND OnResponse fires once for the successful attempt 1 — 2 spans
// started, 2 ended.
func TestRetryAttemptEndsSpan(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"1302","message":"rate limit"}}`)
			return
		}
		writeJSON(w, http.StatusOK, `{"id":"x","model":"m","choices":[]}`)
	}))
	defer srv.Close()

	h := &recordingHook{}
	c := newTestClient(t, srv.URL, Config{MaxRetries: 3, Hooks: []Hook{h}})

	var resp ChatResponse
	if err := c.doRequest(context.Background(), "POST", "/chat/completions", map[string]string{"q": "hi"}, &resp); err != nil {
		t.Fatalf("doRequest: %v", err)
	}
	// 2 OnRequest (one per attempt = 2 spans started).
	if got := len(h.requests); got != 2 {
		t.Fatalf("expected 2 OnRequest (1 per attempt), got %d", got)
	}
	// 1 OnResponse (the success) + 1 OnError (the retried 429) = 2 terminal
	// hooks = 2 spans ended. Before the fix, errors was empty (the retry
	// continue skipped OnError) so only 1 of 2 spans ended.
	if got := len(h.errors); got != 1 {
		t.Fatalf("expected 1 OnError for the retried attempt (span-end); got %d — retry-path span leak", got)
	}
	if got := len(h.responses); got != 1 {
		t.Fatalf("expected 1 OnResponse (success), got %d", got)
	}
	// Total terminal hooks must equal total OnRequest (every started span ends).
	if len(h.errors)+len(h.responses) != len(h.requests) {
		t.Fatalf("span leak: %d started, %d ended (errors+responses)", len(h.requests), len(h.errors)+len(h.responses))
	}
}

// TestHookFiresOnStreamChunk verifies the streaming path fires OnRequest and
// OnStreamChunk for each chunk delivered to the caller, and OnError on
// terminal failure.
func TestHookFiresOnStreamChunk(t *testing.T) {
	srv := httptest.NewServer(sseHandler(
		`{"id":"1","model":"m","choices":[{"index":0,"delta":{"content":"a"}}]}`,
		`{"id":"1","model":"m","choices":[{"index":0,"delta":{"content":"b"}}]}`,
		`[DONE]`,
	))
	defer srv.Close()

	h := &recordingHook{}
	c := newTestClient(t, srv.URL, Config{MaxRetries: 0, Hooks: []Hook{h}})
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
		t.Errorf("expected 2 chunks, got %d", chunks)
	}
	if got := len(h.chunks); got != 2 {
		t.Errorf("expected 2 OnStreamChunk calls, got %d", got)
	}
	if got := len(h.requests); got < 1 {
		t.Errorf("expected >=1 OnRequest call, got %d", got)
	}
	// Service and Model should be stamped by Stream via WithService/WithModel.
	if h.chunks[0].Service != "chat" {
		t.Errorf("expected Service='chat', got %q", h.chunks[0].Service)
	}
	if h.chunks[0].Model != "glm-test" {
		t.Errorf("expected Model='glm-test', got %q", h.chunks[0].Model)
	}
}

// TestHookFiresOnStreamError verifies the streaming path fires OnError when
// the stream fails with a non-retriable error.
func TestHookFiresOnStreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"1308","message":"usage limit"}}`)
	}))
	defer srv.Close()

	h := &recordingHook{}
	c := newTestClient(t, srv.URL, Config{MaxRetries: 0, Hooks: []Hook{h}})
	req := ChatRequest{Model: "glm-test", Messages: []Message{{Role: "user", Content: "hi"}}, TopP: 0.95}

	for _, err := range c.Chat().Stream(context.Background(), req) {
		if err == nil {
			t.Fatal("expected terminal error from stream")
		}
	}
	if got := len(h.errors); got != 1 {
		t.Errorf("expected 1 OnError call, got %d", got)
	}
}

// TestHookCtxStamping verifies WithService/WithModel stamp values that
// buildRequestMeta extracts into RequestMeta for hook consumers.
func TestHookCtxStamping(t *testing.T) {
	ctx := context.Background()
	stamped := WithModel(WithService(ctx, "embeddings"), "embedding-3")
	c := &Client{} // no hooks; buildRequestMeta is independent of hook config

	meta := c.buildRequestMeta(stamped, "POST", "/embeddings", 2)
	if meta.Service != "embeddings" {
		t.Errorf("expected Service='embeddings', got %q", meta.Service)
	}
	if meta.Model != "embedding-3" {
		t.Errorf("expected Model='embedding-3', got %q", meta.Model)
	}
	if meta.Method != "POST" {
		t.Errorf("expected Method='POST', got %q", meta.Method)
	}
	if meta.Endpoint != "/embeddings" {
		t.Errorf("expected Endpoint='/embeddings', got %q", meta.Endpoint)
	}
	if meta.Attempt != 2 {
		t.Errorf("expected Attempt=2, got %d", meta.Attempt)
	}

	// Unstamped context → empty Service/Model.
	plain := c.buildRequestMeta(context.Background(), "GET", "/models", 0)
	if plain.Service != "" || plain.Model != "" {
		t.Errorf("expected empty Service/Model on unstamped ctx, got %+v", plain)
	}
}

// TestHookOnRequestCtxReplaces verifies the context returned by OnRequest is
// the one passed to subsequent hooks and to the actual send. We verify by
// chaining two hooks: the first stamps a value, the second observes it.
// (Context values don't cross the network, so the assertion happens
// in-process via the second hook, not at the server.)
func TestHookOnRequestCtxReplaces(t *testing.T) {
	type ctxKey int
	const myKey ctxKey = 1

	observer := &ctxObservingHook{key: myKey}
	spanHook := &spanAttachingHook{key: myKey, val: "hook-attached"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"id":"x","model":"m","choices":[]}`)
	}))
	defer srv.Close()

	// Order matters: spanHook stamps, then observer reads. Hooks run in slice
	// order, each receiving the prior hook's returned ctx.
	c := newTestClient(t, srv.URL, Config{Hooks: []Hook{spanHook, observer}})

	var resp ChatResponse
	if err := c.doRequest(context.Background(), "POST", "/chat/completions", map[string]string{"q": "hi"}, &resp); err != nil {
		t.Fatalf("doRequest: %v", err)
	}
	if spanHook.attached != 1 {
		t.Errorf("expected 1 OnRequest attach, got %d", spanHook.attached)
	}
	if observer.seen != "hook-attached" {
		t.Errorf("expected second hook to see 'hook-attached' from first, got %q", observer.seen)
	}
}

type spanAttachingHook struct {
	key      any
	val      string
	attached int
}

func (h *spanAttachingHook) OnRequest(ctx context.Context, _ RequestMeta) context.Context {
	h.attached++
	return context.WithValue(ctx, h.key, h.val)
}
func (h *spanAttachingHook) OnResponse(context.Context, ResponseMeta)    {}
func (h *spanAttachingHook) OnError(context.Context, RequestMeta, error) {}
func (h *spanAttachingHook) OnStreamChunk(context.Context, RequestMeta, any) {
}

type ctxObservingHook struct {
	key  any
	seen any
}

func (h *ctxObservingHook) OnRequest(ctx context.Context, _ RequestMeta) context.Context {
	h.seen = ctx.Value(h.key)
	return ctx
}
func (h *ctxObservingHook) OnResponse(context.Context, ResponseMeta)    {}
func (h *ctxObservingHook) OnError(context.Context, RequestMeta, error) {}
func (h *ctxObservingHook) OnStreamChunk(context.Context, RequestMeta, any) {
}

// TestHookGetUsage verifies the GetUsage method exists on ChatResponse and
// returns the embedded Usage as a pointer (the usageBearer contract).
func TestHookGetUsage(t *testing.T) {
	r := &ChatResponse{Usage: Usage{TotalTokens: 42}}
	u := r.GetUsage()
	if u == nil {
		t.Fatal("expected non-nil Usage")
	}
	if u.TotalTokens != 42 {
		t.Errorf("expected 42 tokens, got %d", u.TotalTokens)
	}
	// extractUsage should return the same pointer via the interface.
	if got := extractUsage(r); got != u {
		t.Errorf("extractUsage returned different pointer")
	}
	// Non-usage types → nil.
	if got := extractUsage(nil); got != nil {
		t.Errorf("expected nil for nil result")
	}
	if got := extractUsage("not a response"); got != nil {
		t.Errorf("expected nil for non-usage type")
	}
}

// TestHookGetUsageEmbeddings verifies EmbeddingsResponse satisfies usageBearer
// (Usage is a value type on the wire; GetUsage returns a pointer to it so
// extractUsage surfaces embedding token counts to OnResponse hooks).
func TestHookGetUsageEmbeddings(t *testing.T) {
	r := &EmbeddingsResponse{Usage: Usage{TotalTokens: 7}}
	u := r.GetUsage()
	if u == nil {
		t.Fatal("expected non-nil Usage")
	}
	if u.TotalTokens != 7 {
		t.Errorf("expected 7 tokens, got %d", u.TotalTokens)
	}
	if got := extractUsage(r); got == nil || got.TotalTokens != 7 {
		t.Errorf("extractUsage(EmbeddingsResponse) = %v, want 7 tokens", got)
	}
}

// TestHookGetUsageAsyncResult verifies AsyncResultResponse satisfies
// usageBearer (Usage is already *Usage on the wire; GetUsage returns it
// directly, and nil Usage — e.g. for image/video tasks — stays nil).
func TestHookGetUsageAsyncResult(t *testing.T) {
	// Chat-completion async result carries usage.
	r := &AsyncResultResponse{Usage: &Usage{TotalTokens: 11}}
	u := r.GetUsage()
	if u == nil || u.TotalTokens != 11 {
		t.Errorf("expected 11 tokens, got %v", u)
	}
	if got := extractUsage(r); got != u {
		t.Errorf("extractUsage returned different pointer")
	}
	// Image/video async result has no usage → nil.
	img := &AsyncResultResponse{}
	if got := img.GetUsage(); got != nil {
		t.Errorf("expected nil Usage for image task, got %v", got)
	}
	if got := extractUsage(img); got != nil {
		t.Errorf("expected nil extractUsage for image task, got %v", got)
	}
}

// TestHookDurationRecorded verifies OnResponse carries a positive duration.
func TestHookDurationRecorded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		writeJSON(w, http.StatusOK, `{"id":"x","model":"m","choices":[]}`)
	}))
	defer srv.Close()

	h := &recordingHook{}
	c := newTestClient(t, srv.URL, Config{Hooks: []Hook{h}})

	var resp ChatResponse
	if err := c.doRequest(context.Background(), "POST", "/chat/completions", map[string]string{"q": "hi"}, &resp); err != nil {
		t.Fatalf("doRequest: %v", err)
	}
	if len(h.responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(h.responses))
	}
	if h.responses[0].Duration <= 0 {
		t.Errorf("expected positive duration, got %v", h.responses[0].Duration)
	}
}
