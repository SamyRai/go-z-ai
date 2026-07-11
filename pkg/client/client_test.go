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

// newTestClient builds a client pointed at baseURL with a tiny RetryDelay so
// retry tests stay fast. Tests live in-package to exercise the unexported
// doRequest/doRequestWithContext retry path directly.
func newTestClient(t *testing.T, baseURL string, cfg Config) *Client {
	t.Helper()
	cfg.APIKey = "test-key"
	cfg.BaseURL = baseURL
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = time.Millisecond
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	fmt.Fprint(w, body)
}

// A retriable error (rate-limit code 1302) on the first attempt must be
// retried, then succeed.
func TestRetryOnRetriableThenSuccess(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"1302","message":"Rate limit reached"}}`)
			return
		}
		writeJSON(w, http.StatusOK, `{"id":"x","model":"m","choices":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 3})
	var resp ChatResponse
	if err := c.doRequest("POST", "/chat/completions", map[string]string{"q": "hi"}, &resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls (1 retry), got %d", calls)
	}
	if resp.ID != "x" {
		t.Fatalf("expected resp id 'x', got %q", resp.ID)
	}
}

// A 429 carrying a non-retriable code (quota exhausted 1308) must NOT retry.
func TestNoRetryOnNonRetriableCode(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"1308","message":"Usage limit reached"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 3})
	err := c.doRequest("POST", "/chat/completions", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call (no retry), got %d", calls)
	}
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Code != 1308 {
		t.Fatalf("expected APIError code 1308, got %#v", err)
	}
}

// Retries must stop at MaxRetries and return the last error.
func TestRetryExhaustedReturnsLastError(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		writeJSON(w, http.StatusBadGateway, `{"error":{"code":"-1","message":"bad gateway"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 2})
	err := c.doRequest("POST", "/x", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 3 { // 1 initial + 2 retries
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

// A retriable status (503, code 1305) with a long backoff must abort promptly
// when the context is cancelled, proving the backoff respects cancellation.
func TestRetryBackoffRespectsContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusServiceUnavailable, `{"error":{"code":"1305","message":"overloaded"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 5, RetryDelay: 5 * time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := c.doRequestWithContext(ctx, "POST", "/x", nil, nil)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("backoff should abort on cancel; took %v", elapsed)
	}
}

// A transport failure (connection refused) must be retried and ultimately
// surface as a wrapped transport error rather than hanging.
func TestRetryOnTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := srv.URL
	srv.Close() // free the port → subsequent dials refuse

	c := newTestClient(t, addr, Config{MaxRetries: 2})
	err := c.doRequest("POST", "/x", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to execute request") {
		t.Fatalf("expected wrapped transport error, got %v", err)
	}
}

// retryDelay: a Retry-After header (integer seconds) overrides backoff, capped.
func TestRetryDelayHonorsRetryAfter(t *testing.T) {
	c := newTestClient(t, "http://x", Config{})
	if d := c.retryDelay("2", 5); d != 2*time.Second {
		t.Fatalf("Retry-After should win over backoff; got %v", d)
	}
	if d := c.retryDelay("120", 0); d != maxRetryDelay {
		t.Fatalf("Retry-After should be capped at maxRetryDelay; got %v", d)
	}
}

// retryDelay: exponential backoff = base * 2^attempt + up to 25% jitter.
func TestRetryDelayBackoffBounds(t *testing.T) {
	c := newTestClient(t, "http://x", Config{RetryDelay: 10 * time.Millisecond})
	// attempt 0: 10ms + [0, 2.5ms] jitter → [10ms, 12.5ms]
	d := c.retryDelay("", 0)
	if d < 10*time.Millisecond || d > 13*time.Millisecond {
		t.Fatalf("attempt 0 delay out of expected range: %v", d)
	}
	// Large attempt must clamp to maxRetryDelay (plus up to 25% jitter, so allow headroom).
	dBig := c.retryDelay("", 30)
	if dBig > maxRetryDelay+maxRetryDelay/4 {
		t.Fatalf("large-attempt delay should clamp near maxRetryDelay; got %v", dBig)
	}
}

// MaxRetries == -1 disables retries: a retriable status is attempted once.
func TestMaxRetriesNegativeDisablesRetry(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		writeJSON(w, http.StatusServiceUnavailable, `{"error":{"code":"1305","message":"overloaded"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: -1})
	_ = c.doRequest("POST", "/x", nil, nil)
	if calls != 1 {
		t.Fatalf("MaxRetries=-1 should disable retries; expected 1 call, got %d", calls)
	}
}
