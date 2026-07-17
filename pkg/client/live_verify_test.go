package client

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

// This file verifies the *success*-path response shapes of services that were
// previously modeled only from Z.AI's docs (see docs/roadmap.md "Unverified
// live"). Each test replays a committed cassette offline in CI. To capture a
// cassette against a real entitled account, run one test with recording on:
//
//	ZAI_RECORD=1 ZAI_API_KEY=<real-key> go test -run TestVerifyAnthropicMessages ./pkg/client
//
// The API key is redacted out of the cassette at save time (redactAuth below),
// so a recorded cassette never contains a real credential — confirm with
// `grep -n "Bearer " pkg/client/testdata/cassettes/<name>.yaml` before
// committing (it must read `Bearer REDACTED`). Until a cassette exists, its
// test skips, so CI stays green. A recording that captures a business error
// (e.g. an account not entitled to embeddings) is NOT a verification — don't
// commit it; leave that test skipped.

// redactAuth strips credentials from a recorded request before the cassette is
// written to disk. Runs as a BeforeSaveHook so a real key never reaches the
// committed YAML.
func redactAuth(i *cassette.Interaction) error {
	if i.Request.Headers.Get("Authorization") != "" {
		i.Request.Headers.Set("Authorization", "Bearer REDACTED")
	}
	for _, h := range []string{"X-Api-Key", "Api-Key", "Cookie"} {
		if i.Request.Headers.Get(h) != "" {
			i.Request.Headers.Set(h, "REDACTED")
		}
	}
	return nil
}

// newRecordClient builds a *Client that records live interactions once into the
// named cassette, redacting the credential at save time. Requires ZAI_API_KEY.
func newRecordClient(t *testing.T, cassetteName, baseURL string) *Client {
	t.Helper()
	apiKey := os.Getenv("ZAI_API_KEY")
	if apiKey == "" {
		t.Skip("ZAI_RECORD=1 requires ZAI_API_KEY to record a live cassette")
	}
	r, err := recorder.New(
		filepath.Join("testdata", "cassettes", cassetteName),
		recorder.WithMode(recorder.ModeRecordOnce),
		recorder.WithMatcher(matchMethodAndURL),
		recorder.WithHook(redactAuth, recorder.BeforeSaveHook),
		recorder.WithSkipRequestLatency(true),
	)
	if err != nil {
		t.Fatalf("recorder.New (record): %v", err)
	}
	t.Cleanup(func() { _ = r.Stop() })

	c, err := NewClient(Config{
		APIKey:     apiKey,
		BaseURL:    baseURL,
		HTTPClient: r.GetDefaultClient(),
		MaxRetries: -1,
	})
	if err != nil {
		t.Fatalf("NewClient (record): %v", err)
	}
	return c
}

// newVerifyClient records when ZAI_RECORD=1, otherwise replays. When replaying
// and the cassette hasn't been recorded yet, the test is skipped so CI stays
// green until an entitled account captures it.
func newVerifyClient(t *testing.T, cassetteName, baseURL string) *Client {
	t.Helper()
	if os.Getenv("ZAI_RECORD") == "1" {
		return newRecordClient(t, cassetteName, baseURL)
	}
	path := filepath.Join("testdata", "cassettes", cassetteName+".yaml")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Skipf("cassette %s.yaml not recorded yet — run: ZAI_RECORD=1 ZAI_API_KEY=… go test -run %s ./pkg/client", cassetteName, t.Name())
	}
	return newReplayClient(t, cassetteName, baseURL)
}

// TestVerifyAnthropicMessages confirms the Anthropic-compatible Messages
// success shape (content blocks, stop_reason, usage) and answers the open
// question of whether GLM surfaces reasoning as `thinking` blocks or the
// OpenAI-style `reasoning_content` field — recorded with thinking enabled.
func TestVerifyAnthropicMessages(t *testing.T) {
	c := newVerifyClient(t, "anthropic_messages", AnthropicBaseURL)

	resp, err := c.Anthropic().Create(context.Background(), AnthropicMessageRequest{
		Model:     "glm-4.6",
		MaxTokens: 128,
		Messages:  []AnthropicMessage{AnthropicTextMessage("user", "Reply with the single word: hello")},
		Thinking:  &AnthropicThinking{Type: "enabled", BudgetTokens: 64},
	})
	if err != nil {
		t.Fatalf("Anthropic Create: %v", err)
	}
	if resp.Role != "assistant" {
		t.Errorf("expected role assistant, got %q", resp.Role)
	}
	if resp.Text() == "" {
		t.Error("expected non-empty text content")
	}
	if resp.StopReason == "" {
		t.Error("expected a stop_reason")
	}
	if resp.Usage.OutputTokens <= 0 {
		t.Error("expected output token accounting")
	}
	// Records the reasoning-channel finding for docs/roadmap.md.
	reasonedViaBlock := false
	for _, b := range resp.Content {
		if b.Type == "thinking" {
			reasonedViaBlock = true
		}
	}
	t.Logf("reasoning channel: thinking-block=%v reasoning_content=%q thinking()=%q",
		reasonedViaBlock, resp.ReasoningContent, resp.Thinking())
}

// TestVerifyEmbeddings confirms a real embedding vector parses.
func TestVerifyEmbeddings(t *testing.T) {
	c := newVerifyClient(t, "embeddings_success", BigModelBaseURL)

	resp, err := c.Embeddings().Create(context.Background(), EmbeddingsRequest{
		Model: EmbeddingModel3,
		Input: "the quick brown fox",
	})
	if err != nil {
		t.Fatalf("Embeddings Create: %v", err)
	}
	if len(resp.Data) == 0 || len(resp.Data[0].Embedding) == 0 {
		t.Fatalf("expected a non-empty embedding vector, got %+v", resp.Data)
	}
}

// TestVerifyModerations confirms a real moderation verdict parses.
func TestVerifyModerations(t *testing.T) {
	c := newVerifyClient(t, "moderations_success", BigModelBaseURL)

	resp, err := c.Moderations().Create(context.Background(), ModerationRequest{
		Input: "hello world",
	})
	if err != nil {
		t.Fatalf("Moderations Create: %v", err)
	}
	if len(resp.ResultList) == 0 || resp.ResultList[0].RiskLevel == "" {
		t.Fatalf("expected a moderation result with a risk level, got %+v", resp.ResultList)
	}
}

// TestVerifyAgentsInvoke confirms the Agents Invoke success shape
// (Choices/Usage). AgentID is account-specific — set ZAI_AGENT_ID when
// recording.
func TestVerifyAgentsInvoke(t *testing.T) {
	c := newVerifyClient(t, "agents_invoke_success", AgentsBaseURL)

	agentID := os.Getenv("ZAI_AGENT_ID")
	if os.Getenv("ZAI_RECORD") == "1" && agentID == "" {
		t.Skip("recording Agents Invoke requires ZAI_AGENT_ID (an agent your account owns)")
	}
	if agentID == "" {
		agentID = "replayed-agent" // the recorded cassette pins the real id
	}

	resp, err := c.Agents().Invoke(context.Background(), AgentInvokeRequest{
		AgentID:  agentID,
		Messages: []AgentMessage{NewAgentTextMessage("user", "hello")},
	})
	if err != nil {
		t.Fatalf("Agents Invoke: %v", err)
	}
	if resp.Failed() {
		t.Fatalf("expected a successful agent response, got a business failure: %+v", resp)
	}
	if len(resp.Choices) == 0 {
		t.Error("expected at least one choice in the agent response")
	}
}
