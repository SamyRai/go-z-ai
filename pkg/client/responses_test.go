package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Create posts to the region's /api/v1/responses with the stream flag off,
// sanitized tool schemas, and the reasoning effort; the reply's helpers read
// text, reasoning, function calls, and usage.
func TestResponsesCreate(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/responses" {
			t.Errorf("path = %s, want /api/v1/responses", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("request body: %v", err)
		}
		_, _ = io.WriteString(w, `{"id":"resp_1","object":"response","status":"completed","model":"glm-5.3",
			"output":[
				{"type":"reasoning","summary":[{"type":"summary_text","text":"thinking"}]},
				{"type":"function_call","call_id":"c1","name":"lookup","arguments":"{\"q\":\"x\"}"},
				{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Hel"},{"type":"output_text","text":"lo"}]}],
			"usage":{"input_tokens":5,"output_tokens":7,"total_tokens":12,"output_tokens_details":{"reasoning_tokens":3}}}`)
	}))
	defer srv.Close()

	c := newRedirectingTestClient(t, srv, Config{})
	resp, err := c.Responses().Create(context.Background(), ResponsesRequest{
		Model:        DefaultModel,
		Instructions: "be brief",
		Input:        []ResponsesItem{ResponsesMessage("user", "hi")},
		Reasoning:    &ResponsesReasoning{Effort: EffortHigh},
		Tools: []ResponsesTool{{Name: "lookup", Parameters: map[string]any{
			"type": "object", "properties": map[string]any{"q": map[string]any{"anyOf": []any{
				map[string]any{"type": "string"}, map[string]any{"type": "null"},
			}}},
		}}},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if got["stream"] != nil || got["model"] != DefaultModel || got["instructions"] != "be brief" {
		t.Errorf("request = %v", got)
	}
	if r, _ := got["reasoning"].(map[string]any); r["effort"] != EffortHigh {
		t.Errorf("reasoning = %v", got["reasoning"])
	}
	tool := got["tools"].([]any)[0].(map[string]any)
	if tool["type"] != ToolTypeFunction {
		t.Errorf("tool type defaulted to %v, want function", tool["type"])
	}
	q := tool["parameters"].(map[string]any)["properties"].(map[string]any)["q"].(map[string]any)
	if q["type"] != "string" {
		t.Errorf("tool schema not sanitized: q = %v", q)
	}
	input := got["input"].([]any)[0].(map[string]any)
	if input["role"] != "user" || input["content"].([]any)[0].(map[string]any)["type"] != ResponsesInputText {
		t.Errorf("input = %v", input)
	}

	if resp.OutputText() != "Hello" || resp.ReasoningText() != "thinking" {
		t.Errorf("text=%q reasoning=%q", resp.OutputText(), resp.ReasoningText())
	}
	if calls := resp.FunctionCalls(); len(calls) != 1 || calls[0].CallID != "c1" || calls[0].Arguments != `{"q":"x"}` {
		t.Errorf("function calls = %+v", calls)
	}
	u := resp.GetUsage()
	if u.PromptTokens != 5 || u.CompletionTokens != 7 || u.CompletionTokensDetails.ReasoningTokens != 3 {
		t.Errorf("usage = %+v", u)
	}
}

// A failed response in an HTTP 200 is an *APIError.
func TestResponsesCreateFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"id":"r","status":"failed","error":{"code":"server_error","message":"boom"}}`)
	}))
	defer srv.Close()

	c := newRedirectingTestClient(t, srv, Config{})
	_, err := c.Responses().Create(context.Background(), ResponsesRequest{Model: DefaultModel, Input: []ResponsesItem{ResponsesMessage("user", "hi")}})
	apiErr, ok := errors.AsType[*APIError](err)
	if !ok || !strings.Contains(apiErr.Error(), "boom") {
		t.Fatalf("want *APIError with the server message, got %v", err)
	}
}

func TestValidateResponsesRequest(t *testing.T) {
	msg := []ResponsesItem{ResponsesMessage("user", "hi")}
	for _, tc := range []struct {
		name string
		req  ResponsesRequest
	}{
		{"no model", ResponsesRequest{Input: msg}},
		{"no input", ResponsesRequest{Model: DefaultModel}},
		{"unsupported effort", ResponsesRequest{Model: DefaultModel, Input: msg, Reasoning: &ResponsesReasoning{Effort: EffortMedium}}},
		{"bad tool name", ResponsesRequest{Model: DefaultModel, Input: msg, Tools: []ResponsesTool{{Name: "has space"}}}},
	} {
		if err := validateResponsesRequest(&tc.req); err == nil {
			t.Errorf("%s: expected an error", tc.name)
		}
	}
}

func TestResponsesMessage(t *testing.T) {
	if p := ResponsesMessage("assistant", "x").Content[0]; p.Type != ResponsesOutputText {
		t.Errorf("assistant part type = %s, want output_text", p.Type)
	}
	if p := ResponsesMessage("developer", "x").Content[0]; p.Type != ResponsesInputText {
		t.Errorf("developer part type = %s, want input_text", p.Type)
	}
	if it := ResponsesFunctionOutput("c1", "42"); it.Type != ResponsesItemFunctionCallOutput || it.CallID != "c1" {
		t.Errorf("function output = %+v", it)
	}
}

// The stream yields typed events (type from the payload, else the SSE event
// name) and finishes on response.completed or [DONE].
func TestDecodeResponsesStream(t *testing.T) {
	body := "event: response.output_text.delta\n" +
		"data: {\"type\":\"response.output_text.delta\",\"delta\":\"Hel\"}\n\n" +
		"event: response.output_text.delta\n" +
		"data: {\"delta\":\"lo\"}\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"r\",\"status\":\"completed\",\"usage\":{\"input_tokens\":1,\"output_tokens\":2,\"total_tokens\":3}}}\n\n" +
		"data: [DONE]\n\n"
	var text string
	var final *ResponsesResponse
	err := decodeResponsesStream(context.Background(), strings.NewReader(body), func(ev ResponsesStreamEvent) error {
		switch ev.Type {
		case ResponsesEventOutputTextDelta:
			text += ev.Delta
		case ResponsesEventCompleted:
			final = ev.Response
		}
		return nil
	})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if text != "Hello" || final == nil || final.Usage.TotalTokens != 3 {
		t.Errorf("text=%q final=%+v", text, final)
	}
}

// error and response.failed events end the stream with an *APIError.
func TestDecodeResponsesStreamFailures(t *testing.T) {
	for name, body := range map[string]string{
		"error event":     "data: {\"type\":\"error\",\"code\":\"1113\",\"message\":\"no balance\"}\n\n",
		"failed response": "data: {\"type\":\"response.failed\",\"response\":{\"status\":\"failed\",\"error\":{\"message\":\"no balance\"}}}\n\n",
	} {
		err := decodeResponsesStream(context.Background(), strings.NewReader(body), func(ResponsesStreamEvent) error { return nil })
		apiErr, ok := errors.AsType[*APIError](err)
		if !ok || !strings.Contains(apiErr.Error(), "no balance") {
			t.Errorf("%s: want *APIError with the message, got %v", name, err)
		}
	}
}

// Stream end to end: the request carries stream=true and events arrive in
// order.
func TestResponsesStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if b, _ := io.ReadAll(r.Body); !strings.Contains(string(b), `"stream":true`) {
			t.Error("stream flag not set")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"+
			"data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\"}}\n\n")
	}))
	defer srv.Close()

	c := newRedirectingTestClient(t, srv, Config{})
	var types []string
	err := drainStream(c.Responses().Stream(context.Background(), ResponsesRequest{
		Model: DefaultModel, Input: []ResponsesItem{ResponsesMessage("user", "hi")},
	}), func(ev ResponsesStreamEvent) error {
		types = append(types, ev.Type)
		return nil
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if strings.Join(types, ",") != ResponsesEventOutputTextDelta+","+ResponsesEventCompleted {
		t.Errorf("events = %v", types)
	}
}
