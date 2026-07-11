package client

import (
	"context"
	"net/http"
	"path/filepath"
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
