package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// roundTripFunc adapts a function to http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// newRedirectingTestClient builds a client whose transport redirects every
// outgoing request to srv, regardless of the URL's declared host/scheme —
// the only way to exercise a service against an httptest server when that
// service hits a hardcoded base URL constant (BigModelBaseURL,
// AgentsBaseURL, ...) rather than reading Config.BaseURL.
func newRedirectingTestClient(t *testing.T, srv *httptest.Server, cfg Config) *Client {
	t.Helper()
	target, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", srv.URL, err)
	}
	rt := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req = req.Clone(req.Context())
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host
		return http.DefaultTransport.RoundTrip(req)
	})
	if cfg.APIKey == "" {
		cfg.APIKey = "test-key"
	}
	cfg.HTTPClient = &http.Client{Transport: rt}
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = time.Millisecond
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// newBigModelTestClient builds a client whose transport redirects every
// request to srv — the only way to exercise EmbeddingsService/
// ModerationsService against an httptest server, since both hard-code
// BigModelBaseURL (open.bigmodel.cn) rather than reading Config.BaseURL.
func newBigModelTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	return newRedirectingTestClient(t, srv, Config{ChinaAPIKey: "china-test-key"})
}

// Create posts to /embeddings, authenticates with ChinaAPIKey (not APIKey),
// and parses the response shape confirmed against docs.bigmodel.cn's
// OpenAPI spec.
func TestEmbeddingsCreate(t *testing.T) {
	var gotPath, gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		buf, _ := io.ReadAll(r.Body)
		gotBody = string(buf)
		writeJSON(w, http.StatusOK, `{"object":"list","model":"embedding-3","data":[{"index":0,"object":"embedding","embedding":[0.1,0.2,0.3]}],"usage":{"prompt_tokens":5,"completion_tokens":0,"total_tokens":5}}`)
	}))
	defer srv.Close()

	c := newBigModelTestClient(t, srv)
	resp, err := c.Embeddings().Create(context.Background(), EmbeddingsRequest{
		Model: EmbeddingModel3,
		Input: "你好，今天天气怎么样.",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if gotPath != "/api/paas/v4/embeddings" {
		t.Errorf("expected path /api/paas/v4/embeddings, got %q", gotPath)
	}
	if gotAuth != "Bearer china-test-key" {
		t.Errorf("expected ChinaAPIKey used for auth, got %q", gotAuth)
	}
	if !strings.Contains(gotBody, `"model":"embedding-3"`) {
		t.Errorf("expected model in request body, got: %s", gotBody)
	}
	if len(resp.Data) != 1 || len(resp.Data[0].Embedding) != 3 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Usage.TotalTokens != 5 {
		t.Errorf("expected total_tokens 5, got %d", resp.Usage.TotalTokens)
	}
}

// Create accepts a []string batched Input, marshaled as a JSON array.
func TestEmbeddingsCreateBatchedInput(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		gotBody = string(buf)
		writeJSON(w, http.StatusOK, `{"object":"list","model":"embedding-2","data":[],"usage":{"prompt_tokens":1,"completion_tokens":0,"total_tokens":1}}`)
	}))
	defer srv.Close()

	c := newBigModelTestClient(t, srv)
	_, err := c.Embeddings().Create(context.Background(), EmbeddingsRequest{
		Model: EmbeddingModel2,
		Input: []string{"first", "second"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !strings.Contains(gotBody, `"input":["first","second"]`) {
		t.Errorf("expected array input in request body, got: %s", gotBody)
	}
}

// Missing model or input must fail before any request is sent.
func TestEmbeddingsCreateValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Embeddings().Create(context.Background(), EmbeddingsRequest{Input: "x"}); err == nil {
		t.Error("expected error for missing model")
	}
	if _, err := c.Embeddings().Create(context.Background(), EmbeddingsRequest{Model: EmbeddingModel3}); err == nil {
		t.Error("expected error for missing input")
	}
}

// When ChinaAPIKey is unset, Create falls back to APIKey.
func TestEmbeddingsCreateFallsBackToAPIKey(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		writeJSON(w, http.StatusOK, `{"object":"list","data":[],"usage":{}}`)
	}))
	defer srv.Close()

	c := newRedirectingTestClient(t, srv, Config{APIKey: "only-key"})

	if _, err := c.Embeddings().Create(context.Background(), EmbeddingsRequest{Model: EmbeddingModel3, Input: "x"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if gotAuth != "Bearer only-key" {
		t.Errorf("expected fallback to APIKey, got %q", gotAuth)
	}
}
