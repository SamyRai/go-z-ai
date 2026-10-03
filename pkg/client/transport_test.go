package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Errors whose code the table doesn't know are classified by HTTP status:
// a 404 HTML page or an Anthropic-shaped 400 must not be retried, while an
// unknown 5xx still is.
func TestUnknownErrorsClassifiedByStatus(t *testing.T) {
	cases := []struct {
		name, body string
		status     int
		retriable  bool
		message    string
	}{
		{"html 404", "<html>not found</html>", http.StatusNotFound, false, "<html>not found</html>"},
		{"anthropic 400", `{"type":"error","error":{"type":"invalid_request_error","message":"max_tokens: required"}}`, http.StatusBadRequest, false, "max_tokens: required"},
		{"monitor envelope 401", `{"code":401,"msg":"token expired","success":false}`, http.StatusUnauthorized, false, "token expired"},
		{"empty 502", "", http.StatusBadGateway, true, "Bad Gateway"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := parseAPIError(&http.Response{StatusCode: c.status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(c.body))})
			apiErr, ok := errors.AsType[*APIError](err)
			if !ok {
				t.Fatalf("expected *APIError, got %T", err)
			}
			if apiErr.IsRetriable != c.retriable || apiErr.Message != c.message {
				t.Errorf("got retriable=%v message=%q, want %v %q", apiErr.IsRetriable, apiErr.Message, c.retriable, c.message)
			}
		})
	}
}

// A 404 is attempted once even with retries enabled.
func TestNotFoundIsNotRetried(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c := newTestClient(t, srv.URL, Config{MaxRetries: 3})
	if err := c.doRequest(context.Background(), "GET", "/nope", nil, nil); err == nil {
		t.Fatal("expected an error")
	}
	if calls != 1 {
		t.Errorf("404 attempted %d times, want 1", calls)
	}
}

// Retry-After may be an HTTP date; the delay is capped at maxRetryDelay.
func TestRetryAfterHTTPDate(t *testing.T) {
	c := newTestClient(t, "http://x", Config{})
	soon := time.Now().Add(3 * time.Second).UTC().Format(http.TimeFormat)
	if d := c.retryDelay(soon, 0); d <= 0 || d > 3*time.Second {
		t.Errorf("HTTP-date Retry-After gave %v, want (0, 3s]", d)
	}
	far := time.Now().Add(time.Hour).UTC().Format(http.TimeFormat)
	if d := c.retryDelay(far, 0); d != maxRetryDelay {
		t.Errorf("far Retry-After gave %v, want cap %v", d, maxRetryDelay)
	}
}

// A stream whose first connect attempt is retried must end that attempt's
// span: every OnRequest is paired with exactly one terminal hook.
func TestStreamRetryPairsHooks(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"1302","message":"rate limit"}}`)
			return
		}
		sseHandler(`{"id":"1","model":"m","choices":[{"index":0,"delta":{"content":"x"}}]}`, `[DONE]`)(w, r)
	}))
	defer srv.Close()

	h := &recordingHook{}
	c := newTestClient(t, srv.URL, Config{MaxRetries: 2, Hooks: []Hook{h}})
	req := ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}}
	if err := drainStream(c.Chat().Stream(context.Background(), req), func(StreamChunk) error { return nil }); err != nil {
		t.Fatalf("stream: %v", err)
	}
	if len(h.requests) != 2 || len(h.errors) != 1 || len(h.responses) != 1 {
		t.Fatalf("hooks: %d requests, %d errors, %d responses; want 2/1/1", len(h.requests), len(h.errors), len(h.responses))
	}
	if h.responses[0].Service != "chat" || h.responses[0].Model != "m" {
		t.Errorf("stream attribution = %q/%q, want chat/m", h.responses[0].Service, h.responses[0].Model)
	}
}

// Unary calls carry service/model attribution too, not just streams.
func TestUnaryChatAttribution(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"id":"1","model":"m","choices":[],"usage":{"total_tokens":3}}`)
	}))
	defer srv.Close()
	h := &recordingHook{}
	c := newTestClient(t, srv.URL, Config{Hooks: []Hook{h}})
	if _, err := c.Chat().Create(context.Background(), ChatRequest{Model: "glm-x", Messages: []Message{{Role: "user", Content: "hi"}}}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(h.responses) != 1 || h.responses[0].Service != "chat" || h.responses[0].Model != "glm-x" || h.responses[0].Usage == nil {
		t.Errorf("unexpected response meta: %+v", h.responses)
	}
}

// The Anthropic surface follows Config.Region.
func TestAnthropicFollowsRegion(t *testing.T) {
	tr := &hostRoutes{routes: map[string]stubReply{"open.bigmodel.cn": {status: 200, body: `{"id":"m","type":"message","role":"assistant","content":[]}`}}}
	c, err := NewClient(Config{APIKey: "k", Region: RegionChina, MaxRetries: -1, HTTPClient: &http.Client{Transport: tr}})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = c.Anthropic().Create(context.Background(), AnthropicMessageRequest{
		Model: DefaultModel, MaxTokens: 8, Messages: []AnthropicMessage{AnthropicTextMessage("user", "hi")},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(tr.seen) != 1 || tr.seen[0] != "open.bigmodel.cn" {
		t.Errorf("hosts %v, want open.bigmodel.cn", tr.seen)
	}
}

// Region also picks the default chat base when BaseURL is unset.
func TestDefaultBaseURLFollowsRegion(t *testing.T) {
	c, err := NewClient(Config{APIKey: "k", Region: RegionChina})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.config.BaseURL != BigModelBaseURL {
		t.Errorf("BaseURL = %q, want %q", c.config.BaseURL, BigModelBaseURL)
	}
}
