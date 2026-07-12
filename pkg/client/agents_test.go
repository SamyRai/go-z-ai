package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

// matchMethodAndURL ignores the request body: the cassette was recorded via
// a hand-built map[string]any (alphabetical key order), while our real
// AgentInvokeRequest struct marshals in field-declaration order — same JSON
// content, different bytes. Body-shape correctness is covered by the
// request validation tests instead; this fixture is about response parsing.
func matchMethodAndURL(r *http.Request, i cassette.Request) bool {
	return r.Method == i.Method && r.URL.String() == i.URL
}

// newReplayClient builds a *Client backed by a go-vcr recorder in
// ModeReplayOnly against the named cassette under testdata/cassettes. This
// never touches the network: ModeReplayOnly errors loudly
// (ErrCassetteNotFound/ErrInteractionNotFound) instead of silently falling
// through to a live call if the cassette is missing or incomplete.
//
// Cassette naming convention: one file per endpoint/method under test,
// named after what it calls — "agents_invoke.yaml", "embeddings.yaml",
// "tools_web_search.yaml" — never after when or why it was recorded
// ("new_endpoints_live.yaml" and "tools_live.yaml" were both renamed/split
// away from that pattern — see CHANGELOG.md). If a single interaction doesn't
// stand alone as a meaningful fact on its own (e.g. proving one claim
// requires several endpoints together, like TestBigModelSameKeyAuthenticates
// / bigmodel_same_key.yaml), name the cassette after the claim instead of
// splitting it — but that should be the exception, not the default excuse
// for bundling unrelated calls recorded in the same session.
func newReplayClient(t *testing.T, cassetteName, baseURL string) *Client {
	t.Helper()
	r, err := recorder.New(
		filepath.Join("testdata", "cassettes", cassetteName),
		recorder.WithMode(recorder.ModeReplayOnly),
		recorder.WithMatcher(matchMethodAndURL),
		recorder.WithSkipRequestLatency(true),
	)
	if err != nil {
		t.Fatalf("recorder.New: %v", err)
	}
	t.Cleanup(func() { _ = r.Stop() })

	c, err := NewClient(Config{
		APIKey:     "replayed-from-cassette", // never the real key; cassette has it redacted
		BaseURL:    baseURL,
		HTTPClient: r.GetDefaultClient(),
		MaxRetries: -1, // a replayed error must not trigger a real retry/backoff wait
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// TestAgentsInvokeInsufficientBalance replays a real recorded interaction
// against the live Z.AI Agents API (2026-07-10, account type coding_plan):
// POST https://api.z.ai/api/v1/agents returned HTTP 200 with a business-level
// failure (insufficient balance) encoded in the body — not a non-200 status.
// This is exactly the quirk AgentResponse.Failed exists to handle; replaying
// the real response proves our types parse Z.AI's actual wire format, not
// just a hand-written fixture.
func TestAgentsInvokeInsufficientBalance(t *testing.T) {
	c := newReplayClient(t, "agents_invoke", AgentsBaseURL)

	resp, err := c.Agents().Invoke(context.Background(), AgentInvokeRequest{
		AgentID:  "general_translation",
		Messages: []AgentMessage{NewAgentTextMessage("user", "hello")},
		CustomVariables: map[string]any{
			"source_lang": "auto",
			"target_lang": "zh-CN",
		},
	})
	if err != nil {
		t.Fatalf("Invoke: %v (a business failure must not surface as a Go error — the HTTP status was 200)", err)
	}
	if !resp.Failed() {
		t.Fatal("expected resp.Failed() == true for a recorded insufficient-balance response")
	}
	if resp.AgentID != "general_translation" {
		t.Errorf("expected agent_id general_translation, got %q", resp.AgentID)
	}
	if resp.Status != "failed" {
		t.Errorf("expected status \"failed\", got %q", resp.Status)
	}
	if resp.Error == nil || resp.Error.Code != "1113" {
		t.Errorf("expected error code 1113, got %+v", resp.Error)
	}
}

// AsyncResult posts to /v1/agents/async-result and parses a completed task's
// file-output choices — a different content shape from the synchronous
// Invoke path's plain-text AgentContent.
func TestAgentsAsyncResult(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		buf, _ := io.ReadAll(r.Body)
		gotBody = string(buf)
		writeJSON(w, http.StatusOK, `{"agent_id":"intelligent_education_correction_polling","async_id":"async-1","status":"success","choices":[{"messages":[{"role":"assistant","content":[{"type":"file_url","file_url":"https://example.com/result.pdf","tag_cn":"批改结果","tag_en":"Correction Result"}]}]}],"usage":{"total_tokens":42}}`)
	}))
	defer srv.Close()

	// AgentsBaseURL is a hardcoded constant, not derived from Config.BaseURL
	// (see AgentsBaseURL's doc comment), so newTestClient's srv.URL override
	// wouldn't be used — redirect the transport instead, like
	// newBigModelTestClient does for BigModelBaseURL.
	c := newRedirectingTestClient(t, srv, Config{})
	resp, err := c.Agents().AsyncResult(context.Background(), AgentAsyncResultRequest{
		AgentID: "intelligent_education_correction_polling",
		AsyncID: "async-1",
	})
	if err != nil {
		t.Fatalf("AsyncResult: %v", err)
	}
	if gotPath != "/api/v1/agents/async-result" {
		t.Errorf("expected path /api/v1/agents/async-result, got %q", gotPath)
	}
	if !strings.Contains(gotBody, `"async_id":"async-1"`) {
		t.Errorf("expected async_id in request body, got: %s", gotBody)
	}
	if !resp.Done() {
		t.Error("expected Done() == true for status \"success\"")
	}
	if len(resp.Choices) != 1 || resp.Choices[0].Messages[0].Content[0].FileURL != "https://example.com/result.pdf" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 42 {
		t.Errorf("unexpected usage: %+v", resp.Usage)
	}
}

// A "pending" status must not be reported as Done.
func TestAgentAsyncResultResponseDone(t *testing.T) {
	for status, want := range map[string]bool{
		AgentAsyncStatusPending: false,
		AgentAsyncStatusSuccess: true,
		AgentAsyncStatusFailed:  true,
	} {
		r := &AgentAsyncResultResponse{Status: status}
		if got := r.Done(); got != want {
			t.Errorf("status %q: Done() = %v, want %v", status, got, want)
		}
	}
}

// Missing agent_id or async_id must fail before any request is sent.
func TestAgentsAsyncResultValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Agents().AsyncResult(context.Background(), AgentAsyncResultRequest{AsyncID: "a"}); err == nil {
		t.Error("expected error for missing agent_id")
	}
	if _, err := c.Agents().AsyncResult(context.Background(), AgentAsyncResultRequest{AgentID: "a"}); err == nil {
		t.Error("expected error for missing async_id")
	}
}
