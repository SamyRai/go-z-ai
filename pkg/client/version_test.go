package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestUserAgentDefault verifies the client sends the default
// "go-z-ai/<version>" User-Agent on every request. This is the compliance
// hygiene that distinguishes go-z-ai from prohibited SDK access under Z.AI's
// coding-endpoint usage policy (three violations = account ban). See
// docs/en/coding-tools.md and https://docs.z.ai/devpack/usage-policy.
func TestUserAgentDefault(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		writeJSON(w, http.StatusOK, `{"id":"x","model":"m","choices":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{})
	var resp ChatResponse
	if err := c.doRequest(context.Background(), "POST", "/chat/completions", map[string]string{"q": "hi"}, &resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(gotUA, "go-z-ai/") {
		t.Errorf("default User-Agent = %q, want prefix %q", gotUA, "go-z-ai/")
	}
	// The version segment must be non-empty (either a real version from ldflags
	// or the "dev" default for `go build` from source).
	rest := strings.TrimPrefix(gotUA, "go-z-ai/")
	if rest == "" {
		t.Errorf("default User-Agent = %q, want non-empty version segment", gotUA)
	}
}

// TestUserAgentOverride verifies Config.UserAgent replaces the default. This
// matters for downstream apps, proxies, and MCP servers that need to identify
// themselves distinctly while still routing through go-z-ai.
func TestUserAgentOverride(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		writeJSON(w, http.StatusOK, `{"id":"x","model":"m","choices":[]}`)
	}))
	defer srv.Close()

	const custom = "my-proxy/1.2.3 (contact=ops@example.com)"
	c := newTestClient(t, srv.URL, Config{UserAgent: custom})
	var resp ChatResponse
	if err := c.doRequest(context.Background(), "POST", "/chat/completions", map[string]string{"q": "hi"}, &resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotUA != custom {
		t.Errorf("overridden User-Agent = %q, want %q", gotUA, custom)
	}
}

// TestMultipartUserAgentAndNoRetry verifies a multipart upload goes through
// the shared transport: it carries the User-Agent and its content type, and a
// retriable failure is not retried (re-uploading is the caller's decision).
func TestMultipartUserAgentAndNoRetry(t *testing.T) {
	var gotUA, gotType string
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		gotUA, gotType = r.Header.Get("User-Agent"), r.Header.Get("Content-Type")
		writeJSON(w, http.StatusServiceUnavailable, `{"error":{"code":"1305","message":"overloaded"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 3})
	form, err := newMultipartBody(&formFile{field: "file", name: "a.wav", data: []byte("x")}, "model", "glm-asr-2512")
	if err != nil {
		t.Fatalf("newMultipartBody: %v", err)
	}
	if err := c.do(context.Background(), apiRequest{method: "POST", path: "/audio/transcriptions", form: form}, nil); err == nil {
		t.Fatal("expected the 503 to surface")
	}
	if calls != 1 {
		t.Errorf("multipart upload attempted %d times, want 1 (no retry)", calls)
	}
	if !strings.HasPrefix(gotUA, "go-z-ai/") {
		t.Errorf("User-Agent = %q, want prefix go-z-ai/", gotUA)
	}
	if !strings.HasPrefix(gotType, "multipart/form-data; boundary=") {
		t.Errorf("Content-Type = %q, want multipart/form-data", gotType)
	}
}

// TestVersionNonEmpty locks in that Version() returns a non-empty string for
// feature-detection / telemetry uses (it's either the ldflags-injected
// version or "dev" for from-source builds).
func TestVersionNonEmpty(t *testing.T) {
	if v := Version(); v == "" {
		t.Errorf("Version() = %q, want non-empty", v)
	}
}

// TestUserAgentStringMatchesVersion locks the relationship between the
// package-level UserAgent() helper and Version() — callers reusing
// UserAgent() for their own requests must see a consistent value.
func TestUserAgentStringMatchesVersion(t *testing.T) {
	ua := UserAgent()
	wantPrefix := "go-z-ai/" + Version()
	if !strings.HasPrefix(ua, wantPrefix) {
		t.Errorf("UserAgent() = %q, want prefix %q", ua, wantPrefix)
	}
}
