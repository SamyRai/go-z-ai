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
// doRequest retry path directly.
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
	if err := c.doRequest(context.Background(), "POST", "/chat/completions", map[string]string{"q": "hi"}, &resp); err != nil {
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
	err := c.doRequest(context.Background(), "POST", "/chat/completions", nil, nil)
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
	err := c.doRequest(context.Background(), "POST", "/x", nil, nil)
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
	err := c.doRequest(ctx, "POST", "/x", nil, nil)
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
	err := c.doRequest(context.Background(), "POST", "/x", nil, nil)
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
	_ = c.doRequest(context.Background(), "POST", "/x", nil, nil)
	if calls != 1 {
		t.Fatalf("MaxRetries=-1 should disable retries; expected 1 call, got %d", calls)
	}
}

// The Authorization header must be stripped when a redirect crosses to a
// different host, so a compromised/misconfigured upstream or transparent proxy
// can't capture the bearer token. Go's net/http default only strips it on a
// scheme change, not a host change.
func TestRedirectStripsAuthCrossHost(t *testing.T) {
	var leakedAuth string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		leakedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Different host (the target's 127.0.0.1:<port>) → triggers the
		// cross-host CheckRedirect branch.
		http.Redirect(w, r, target.URL+"/sink", http.StatusFound)
	}))
	defer origin.Close()

	c, err := NewClient(Config{APIKey: "secret-key-xyz", BaseURL: origin.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// Hit the origin directly so the client follows the 302 to the target.
	// Use a raw http.Get-style call through the client's configured transport
	// by issuing a real request via doRequest against the origin's base URL.
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, origin.URL+"/start", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer secret-key-xyz")
	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()
	resp.Body.Close() // drain promptly

	if leakedAuth != "" {
		t.Fatalf("Authorization header leaked to cross-host redirect target: %q", leakedAuth)
	}
}

// Same-host redirects must keep the Authorization header (the token is still
// valid for the original host).
func TestRedirectKeepsAuthSameHost(t *testing.T) {
	var seenAuth string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	// Re-route /redirect to the same target via a handler on the same server
	// so host doesn't change.
	mux := http.NewServeMux()
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/sink", http.StatusFound)
	})
	mux.HandleFunc("/sink", func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	})
	origin := httptest.NewServer(mux)
	defer origin.Close()

	c, err := NewClient(Config{APIKey: "secret-key-xyz", BaseURL: origin.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, origin.URL+"/redirect", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer secret-key-xyz")
	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()

	if seenAuth != "Bearer secret-key-xyz" {
		t.Fatalf("Authorization header should be preserved on same-host redirect; got %q", seenAuth)
	}
}
