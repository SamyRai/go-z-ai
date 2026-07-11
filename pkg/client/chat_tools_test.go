package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Happy path: model requests one tool call, the loop executes it, then the
// model returns a final answer. The 2nd request's wire format is verified to
// carry the echoed assistant tool_calls message and the role:tool result.
func TestRunWithToolsHappyPath(t *testing.T) {
	calls := 0
	var lastReqBody []byte
	var seenName, seenArgs string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		lastReqBody = body
		if calls == 1 {
			writeJSON(w, http.StatusOK, `{"id":"1","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"SF\"}"}}]},"finish_reason":"tool_calls"}]}`)
			return
		}
		writeJSON(w, http.StatusOK, `{"id":"1","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"It is sunny"},"finish_reason":"stop"}]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	req := ChatRequest{
		Model:    "m",
		Messages: []Message{{Role: "user", Content: "weather in SF?"}},
		TopP:     0.95,
		Tools: []Tool{{
			Type: "function",
			Function: &FunctionDef{
				Name: "get_weather", Description: "weather",
				Parameters: map[string]interface{}{"type": "object"},
			},
		}},
	}

	resp, err := c.Chat().RunWithTools(context.Background(), req, func(name, args string) (string, error) {
		seenName, seenArgs = name, args
		return `{"temp_f":72,"condition":"sunny"}`, nil
	})
	if err != nil {
		t.Fatalf("RunWithTools: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
	if seenName != "get_weather" || seenArgs != `{"city":"SF"}` {
		t.Fatalf("executor saw name=%q args=%q", seenName, seenArgs)
	}
	if resp.Choices[0].Message.Content != "It is sunny" {
		t.Fatalf("expected final content 'It is sunny', got %q", resp.Choices[0].Message.Content)
	}

	var sent struct {
		Messages []struct {
			Role       string `json:"role"`
			ToolCallID string `json:"tool_call_id,omitempty"`
			Content    string `json:"content"`
			ToolCalls  []struct {
				ID string `json:"id"`
			} `json:"tool_calls,omitempty"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(lastReqBody, &sent); err != nil {
		t.Fatalf("unmarshal 2nd request: %v", err)
	}
	var sawAssistant, sawTool bool
	for _, m := range sent.Messages {
		if m.Role == "assistant" && len(m.ToolCalls) > 0 && m.ToolCalls[0].ID == "call_1" {
			sawAssistant = true
		}
		if m.Role == "tool" && m.ToolCallID == "call_1" && m.Content != "" {
			sawTool = true
		}
	}
	if !sawAssistant {
		t.Error("2nd request missing echoed assistant tool_calls message")
	}
	if !sawTool {
		t.Error("2nd request missing role:tool result with tool_call_id")
	}
}

// Declaring no tools must error immediately rather than calling the API.
func TestRunWithToolsNoToolsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected API call")
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	req := ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}, TopP: 0.95}
	if _, err := c.Chat().RunWithTools(context.Background(), req, func(string, string) (string, error) {
		return "", nil
	}); err == nil {
		t.Fatal("expected error when no tools declared")
	}
}

// An executor error must be recorded as the tool's content so the model can react.
func TestRunWithToolsExecutorError(t *testing.T) {
	calls := 0
	var toolContent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		if calls == 2 {
			var sent struct {
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
			}
			_ = json.Unmarshal(body, &sent)
			for _, m := range sent.Messages {
				if m.Role == "tool" {
					toolContent = m.Content
				}
			}
			writeJSON(w, http.StatusOK, `{"id":"1","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"sorry"},"finish_reason":"stop"}]}`)
			return
		}
		writeJSON(w, http.StatusOK, `{"id":"1","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"boom","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	req := ChatRequest{
		Model: "m", TopP: 0.95,
		Messages: []Message{{Role: "user", Content: "do it"}},
		Tools:    []Tool{{Type: "function", Function: &FunctionDef{Name: "boom"}}},
	}
	if _, err := c.Chat().RunWithTools(context.Background(), req, func(string, string) (string, error) {
		return "", fmt.Errorf("kaboom")
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(toolContent, "error:") {
		t.Fatalf("expected tool content to record executor error, got %q", toolContent)
	}
}

// A model that never stops requesting tools must hit the round limit.
func TestRunWithToolsRoundLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"id":"1","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"","tool_calls":[{"id":"c","type":"function","function":{"name":"loop","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	req := ChatRequest{
		Model: "m", TopP: 0.95,
		Messages: []Message{{Role: "user", Content: "loop"}},
		Tools:    []Tool{{Type: "function", Function: &FunctionDef{Name: "loop"}}},
	}
	if _, err := c.Chat().RunWithToolsLimit(context.Background(), req, func(string, string) (string, error) {
		return "ok", nil
	}, 2); err == nil {
		t.Fatal("expected round-limit error")
	}
}
