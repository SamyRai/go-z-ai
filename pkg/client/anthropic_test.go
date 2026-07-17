package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Create posts to /v1/messages with Bearer auth and the anthropic-version
// header, sends the typed request body, and parses a message response
// (text + tool_use blocks, stop_reason, usage).
func TestAnthropicCreate(t *testing.T) {
	var gotPath, gotAuth, gotVersion, gotAccept, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotVersion = r.Header.Get("anthropic-version")
		gotAccept = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		writeJSON(w, http.StatusOK, `{
			"id":"msg_1","type":"message","role":"assistant","model":"glm-4.6",
			"content":[
				{"type":"text","text":"Hello "},
				{"type":"text","text":"there"},
				{"type":"tool_use","id":"tu_1","name":"get_weather","input":{"city":"SF"}}
			],
			"stop_reason":"tool_use","stop_sequence":null,
			"usage":{"input_tokens":12,"output_tokens":7}
		}`)
	}))
	defer srv.Close()

	c := newRedirectingTestClient(t, srv, Config{})
	temp := 0.5
	resp, err := c.Anthropic().Create(context.Background(), AnthropicMessageRequest{
		Model:       "glm-4.6",
		MaxTokens:   256,
		System:      "be brief",
		Temperature: &temp,
		Messages:    []AnthropicMessage{AnthropicTextMessage("user", "weather in SF?")},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// The redirecting transport preserves the full path, so this confirms the
	// request URL is AnthropicBaseURL ("…/api/anthropic") + "/v1/messages".
	if gotPath != "/api/anthropic/v1/messages" {
		t.Errorf("path = %q, want /api/anthropic/v1/messages", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("auth = %q, want Bearer test-key", gotAuth)
	}
	if gotVersion != AnthropicVersion {
		t.Errorf("anthropic-version = %q, want %q", gotVersion, AnthropicVersion)
	}
	if gotAccept != "application/json" {
		t.Errorf("content-type = %q", gotAccept)
	}
	if !strings.Contains(gotBody, `"max_tokens":256`) || !strings.Contains(gotBody, `"system":"be brief"`) {
		t.Errorf("request body missing fields: %s", gotBody)
	}
	if !strings.Contains(gotBody, `"temperature":0.5`) {
		t.Errorf("temperature not sent: %s", gotBody)
	}

	if resp.Text() != "Hello there" {
		t.Errorf("Text() = %q, want %q", resp.Text(), "Hello there")
	}
	if resp.StopReason != "tool_use" {
		t.Errorf("stop_reason = %q", resp.StopReason)
	}
	if resp.Usage.InputTokens != 12 || resp.Usage.OutputTokens != 7 {
		t.Errorf("usage = %+v", resp.Usage)
	}
	if len(resp.Content) != 3 || resp.Content[2].Type != "tool_use" || resp.Content[2].Name != "get_weather" {
		t.Fatalf("unexpected content blocks: %+v", resp.Content)
	}
	var input map[string]any
	if err := json.Unmarshal(resp.Content[2].Input, &input); err != nil {
		t.Fatalf("tool_use input not JSON: %v", err)
	}
	if input["city"] != "SF" {
		t.Errorf("tool_use input = %v", input)
	}
}

// Unset optional sampling params (Temperature/TopP/TopK nil) are omitted, not
// sent as zero — sending temperature:0 would silently change behavior.
func TestAnthropicCreateOmitsUnsetSampling(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		writeJSON(w, http.StatusOK, `{"id":"m","type":"message","role":"assistant","content":[],"usage":{"input_tokens":1,"output_tokens":1}}`)
	}))
	defer srv.Close()

	c := newRedirectingTestClient(t, srv, Config{})
	if _, err := c.Anthropic().Create(context.Background(), AnthropicMessageRequest{
		Model:     "glm-4.6",
		MaxTokens: 16,
		Messages:  []AnthropicMessage{AnthropicTextMessage("user", "hi")},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	for _, field := range []string{"temperature", "top_p", "top_k", "stream", "system", "tools"} {
		if strings.Contains(gotBody, `"`+field+`"`) {
			t.Errorf("unset field %q should be omitted, got: %s", field, gotBody)
		}
	}
}

// Tool input_schema is flattened for GLM by default, and left raw when the
// caller opts out.
func TestAnthropicCreateFlattensToolInputSchema(t *testing.T) {
	capture := func(cfg Config) map[string]any {
		var body []byte
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ = io.ReadAll(r.Body)
			writeJSON(w, http.StatusOK, `{"id":"m","type":"message","role":"assistant","content":[],"usage":{"input_tokens":1,"output_tokens":1}}`)
		}))
		defer srv.Close()

		c := newRedirectingTestClient(t, srv, cfg)
		var raw map[string]any
		if err := json.Unmarshal([]byte(`{"type":"object","properties":{"q":{"anyOf":[{"type":"string"},{"type":"null"}]}}}`), &raw); err != nil {
			t.Fatal(err)
		}
		req := AnthropicMessageRequest{
			Model: "glm-4.6", MaxTokens: 16,
			Messages: []AnthropicMessage{AnthropicTextMessage("user", "hi")},
			Tools:    []AnthropicTool{{Name: "lookup", InputSchema: raw}},
		}
		if _, err := c.Anthropic().Create(context.Background(), req); err != nil {
			t.Fatalf("Create: %v", err)
		}
		var sent struct {
			Tools []struct {
				InputSchema map[string]any `json:"input_schema"`
			} `json:"tools"`
		}
		if err := json.Unmarshal(body, &sent); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		return sent.Tools[0].InputSchema["properties"].(map[string]any)["q"].(map[string]any)
	}

	def := capture(Config{})
	if _, ok := def["anyOf"]; ok {
		t.Error("default should flatten input_schema anyOf")
	}
	if def["type"] != "string" {
		t.Errorf("expected flattened type string, got %v", def["type"])
	}

	raw := capture(Config{DisableToolSchemaCompat: true})
	if _, ok := raw["anyOf"]; !ok {
		t.Error("opt-out should keep raw input_schema anyOf")
	}
}

// A thinking-enabled request serializes the thinking config, and a response
// with a thinking block is exposed via Thinking() while Text() stays clean.
func TestAnthropicThinking(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		writeJSON(w, http.StatusOK, `{
			"id":"m","type":"message","role":"assistant","model":"glm-4.6",
			"content":[
				{"type":"thinking","thinking":"Let me reason.","signature":"sig"},
				{"type":"text","text":"Answer."}
			],
			"stop_reason":"end_turn","usage":{"input_tokens":3,"output_tokens":4}
		}`)
	}))
	defer srv.Close()

	c := newRedirectingTestClient(t, srv, Config{})
	resp, err := c.Anthropic().Create(context.Background(), AnthropicMessageRequest{
		Model: "glm-4.6", MaxTokens: 1024,
		Messages: []AnthropicMessage{AnthropicTextMessage("user", "hi")},
		Thinking: &AnthropicThinking{Type: "enabled", BudgetTokens: 512},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !strings.Contains(gotBody, `"thinking":{"type":"enabled","budget_tokens":512}`) {
		t.Errorf("thinking config not serialized: %s", gotBody)
	}
	if resp.Thinking() != "Let me reason." {
		t.Errorf("Thinking() = %q", resp.Thinking())
	}
	if resp.Text() != "Answer." {
		t.Errorf("Text() = %q (thinking should not leak into Text)", resp.Text())
	}
}

// When the endpoint returns no thinking block but does return the OpenAI-style
// reasoning_content field (the GLM/claude-code-router#1133 case), Thinking()
// falls back to it.
func TestAnthropicThinkingReasoningContentFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{
			"id":"m","type":"message","role":"assistant",
			"content":[{"type":"text","text":"Answer."}],
			"reasoning_content":"raw reasoning",
			"usage":{"input_tokens":1,"output_tokens":1}
		}`)
	}))
	defer srv.Close()

	c := newRedirectingTestClient(t, srv, Config{})
	resp, err := c.Anthropic().Create(context.Background(), AnthropicMessageRequest{
		Model: "glm-4.6", MaxTokens: 16,
		Messages: []AnthropicMessage{AnthropicTextMessage("user", "hi")},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resp.Thinking() != "raw reasoning" {
		t.Errorf("Thinking() fallback = %q, want %q", resp.Thinking(), "raw reasoning")
	}
}

// Request validation catches the required-field mistakes before any HTTP call.
func TestAnthropicCreateValidation(t *testing.T) {
	c := newTestClient(t, "http://x", Config{MaxRetries: -1})
	cases := map[string]AnthropicMessageRequest{
		"missing model":   {MaxTokens: 8, Messages: []AnthropicMessage{AnthropicTextMessage("user", "hi")}},
		"no messages":     {Model: "m", MaxTokens: 8},
		"zero max_tokens": {Model: "m", Messages: []AnthropicMessage{AnthropicTextMessage("user", "hi")}},
		"no user message": {Model: "m", MaxTokens: 8, Messages: []AnthropicMessage{AnthropicTextMessage("assistant", "hi")}},
	}
	for name, req := range cases {
		if _, err := c.Anthropic().Create(context.Background(), req); err == nil {
			t.Errorf("%s: expected validation error", name)
		}
	}
}

// CreateStream parses Anthropic's event:/data: SSE framing into typed events,
// skips ping/comment lines, and stops when onEvent returns an error.
func TestAnthropicCreateStream(t *testing.T) {
	stream := "event: message_start\n" +
		"data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\"}}\n" +
		"\n" +
		": ping keep-alive\n" +
		"event: content_block_delta\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hi\"}}\n" +
		"\n" +
		"event: message_stop\n" +
		"data: {\"type\":\"message_stop\"}\n" +
		"\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(func() string { b, _ := io.ReadAll(r.Body); return string(b) }(), `"stream":true`) {
			t.Error("stream flag not set on request")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, stream)
	}))
	defer srv.Close()

	c := newRedirectingTestClient(t, srv, Config{})
	var types []string
	var text strings.Builder
	err := c.Anthropic().CreateStream(context.Background(), AnthropicMessageRequest{
		Model: "glm-4.6", MaxTokens: 32,
		Messages: []AnthropicMessage{AnthropicTextMessage("user", "hi")},
	}, func(ev AnthropicStreamEvent) error {
		types = append(types, ev.Type)
		if ev.Type == "content_block_delta" {
			var d struct {
				Delta struct {
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal(ev.Data, &d); err != nil {
				return err
			}
			text.WriteString(d.Delta.Text)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("CreateStream: %v", err)
	}
	want := []string{"message_start", "content_block_delta", "message_stop"}
	if strings.Join(types, ",") != strings.Join(want, ",") {
		t.Errorf("events = %v, want %v", types, want)
	}
	if text.String() != "Hi" {
		t.Errorf("accumulated text = %q, want Hi", text.String())
	}
}

// Multiple data: lines within one SSE event are joined with newlines so the
// combined payload stays valid (per the SSE spec).
func TestAnthropicCreateStreamMultilineData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// A JSON object split across two data: lines.
		_, _ = io.WriteString(w, "event: x\ndata: {\"a\":1,\ndata: \"b\":2}\n\n")
	}))
	defer srv.Close()

	c := newRedirectingTestClient(t, srv, Config{})
	var gotData string
	err := c.Anthropic().CreateStream(context.Background(), AnthropicMessageRequest{
		Model: "glm-4.6", MaxTokens: 8,
		Messages: []AnthropicMessage{AnthropicTextMessage("user", "hi")},
	}, func(ev AnthropicStreamEvent) error {
		gotData = string(ev.Data)
		return nil
	})
	if err != nil {
		t.Fatalf("CreateStream: %v", err)
	}
	var obj map[string]int
	if err := json.Unmarshal([]byte(gotData), &obj); err != nil {
		t.Fatalf("joined data is not valid JSON (%q): %v", gotData, err)
	}
	if obj["a"] != 1 || obj["b"] != 2 {
		t.Errorf("joined object = %v, want a=1 b=2", obj)
	}
}

// An error returned by onEvent aborts the stream and propagates out.
func TestAnthropicCreateStreamAbort(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: a\ndata: {}\n\nevent: b\ndata: {}\n\n")
	}))
	defer srv.Close()

	c := newRedirectingTestClient(t, srv, Config{})
	seen := 0
	err := c.Anthropic().CreateStream(context.Background(), AnthropicMessageRequest{
		Model: "glm-4.6", MaxTokens: 8,
		Messages: []AnthropicMessage{AnthropicTextMessage("user", "hi")},
	}, func(ev AnthropicStreamEvent) error {
		seen++
		return io.EOF // abort on the first event
	})
	if err == nil {
		t.Fatal("expected error to propagate from onEvent")
	}
	if seen != 1 {
		t.Errorf("expected abort after 1 event, saw %d", seen)
	}
}
