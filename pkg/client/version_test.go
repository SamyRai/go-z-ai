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

// TestUserAgentOnMultipart verifies sendMultipart (used by Audio.Transcribe)
// also carries the User-Agent — covers the second send path independently.
func TestUserAgentOnMultipart(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		writeJSON(w, http.StatusOK, `{}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{})
	if _, err := c.sendMultipart(context.Background(), "/audio/transcriptions", "multipart/form-data; boundary=x", []byte("body")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(gotUA, "go-z-ai/") {
		t.Errorf("sendMultipart User-Agent = %q, want prefix %q", gotUA, "go-z-ai/")
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
