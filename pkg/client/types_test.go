package client

import (
	"encoding/json"
	"strings"
	"testing"
)

// NewJSONSchemaFormat must marshal to the Z.AI structured-output wire shape.
func TestResponseFormatJSONSchemaMarshal(t *testing.T) {
	rf := NewJSONSchemaFormat("cities", json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`), true)
	out, err := json.Marshal(rf)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(out)
	for _, want := range []string{
		`"type":"json_schema"`,
		`"name":"cities"`,
		`"strict":true`,
		`"schema":{"type":"object"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("expected %q in %s", want, s)
		}
	}
}

// A tool-result Message must serialize tool_call_id and name.
func TestMessageToolFieldsMarshal(t *testing.T) {
	m := Message{Role: "tool", ToolCallID: "call_1", Name: "get_weather", Content: `{"temp":72}`}
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(out)
	for _, want := range []string{`"role":"tool"`, `"tool_call_id":"call_1"`, `"name":"get_weather"`} {
		if !strings.Contains(s, want) {
			t.Errorf("expected %q in %s", want, s)
		}
	}
}

// A plain user message must keep its original wire shape (role + content).
func TestMessagePlainMarshal(t *testing.T) {
	m := Message{Role: "user", Content: "hi"}
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, `"role":"user"`) || !strings.Contains(s, `"content":"hi"`) {
		t.Errorf("plain message shape changed: %s", s)
	}
	if strings.Contains(s, "tool_calls") || strings.Contains(s, "tool_call_id") {
		t.Errorf("plain message should not emit tool fields: %s", s)
	}
}

// A message with Images must marshal Content as a content-parts array
// (text part first, then one image_url part per image) instead of a string.
func TestMessageImagesMarshal(t *testing.T) {
	m := Message{Role: "user", Content: "what's in this image?", Images: []string{"https://example.com/a.png", "data:image/png;base64,abcd"}}
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded struct {
		Role    string `json:"role"`
		Content []struct {
			Type     string `json:"type"`
			Text     string `json:"text,omitempty"`
			ImageURL struct {
				URL string `json:"url"`
			} `json:"image_url,omitempty"`
		} `json:"content"`
	}
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("re-unmarshal into wire shape: %v (raw: %s)", err, out)
	}
	if len(decoded.Content) != 3 {
		t.Fatalf("expected 3 content parts (1 text + 2 images), got %d: %s", len(decoded.Content), out)
	}
	if decoded.Content[0].Type != "text" || decoded.Content[0].Text != "what's in this image?" {
		t.Errorf("expected first part to be the text, got %+v", decoded.Content[0])
	}
	if decoded.Content[1].Type != "image_url" || decoded.Content[1].ImageURL.URL != "https://example.com/a.png" {
		t.Errorf("expected second part to be the first image, got %+v", decoded.Content[1])
	}
	if decoded.Content[2].ImageURL.URL != "data:image/png;base64,abcd" {
		t.Errorf("expected third part to be the second image, got %+v", decoded.Content[2])
	}
}

// A message with Images but no text content omits the text part entirely.
func TestMessageImagesOnlyNoTextPart(t *testing.T) {
	m := Message{Role: "user", Images: []string{"https://example.com/a.png"}}
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(out), `"type":"text"`) {
		t.Errorf("expected no text part when Content is empty, got: %s", out)
	}
}

// Unmarshaling must round-trip both wire shapes: plain string content, and
// a content-parts array (reconstituting Images from image_url parts).
func TestMessageUnmarshalRoundTrip(t *testing.T) {
	plain := Message{Role: "user", Content: "hi", Name: "n"}
	data, err := json.Marshal(plain)
	if err != nil {
		t.Fatalf("marshal plain: %v", err)
	}
	var gotPlain Message
	if err := json.Unmarshal(data, &gotPlain); err != nil {
		t.Fatalf("unmarshal plain: %v", err)
	}
	if gotPlain.Role != plain.Role || gotPlain.Content != plain.Content || gotPlain.Name != plain.Name || len(gotPlain.Images) != 0 {
		t.Errorf("plain round-trip mismatch: got %+v, want %+v", gotPlain, plain)
	}

	multi := Message{Role: "user", Content: "look", Images: []string{"https://x/a.png", "https://x/b.png"}}
	data, err = json.Marshal(multi)
	if err != nil {
		t.Fatalf("marshal multimodal: %v", err)
	}
	var gotMulti Message
	if err := json.Unmarshal(data, &gotMulti); err != nil {
		t.Fatalf("unmarshal multimodal: %v", err)
	}
	if gotMulti.Content != "look" {
		t.Errorf("expected text content 'look', got %q", gotMulti.Content)
	}
	if len(gotMulti.Images) != 2 || gotMulti.Images[0] != "https://x/a.png" || gotMulti.Images[1] != "https://x/b.png" {
		t.Errorf("expected 2 images round-tripped, got %v", gotMulti.Images)
	}
}

// NewFunctionTool must emit the documented function tool wire shape.
func TestFunctionToolMarshal(t *testing.T) {
	tool := NewFunctionTool("get_weather", "weather lookup", map[string]any{
		"type":       "object",
		"properties": map[string]any{"city": map[string]any{"type": "string"}},
		"required":   []any{"city"},
	})
	out, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(out)
	for _, want := range []string{
		`"type":"function"`,
		`"function":{`,
		`"name":"get_weather"`,
		`"description":"weather lookup"`,
		`"parameters":{`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("expected %q in %s", want, s)
		}
	}
	// A function tool must not emit retrieval/web_search payloads.
	if strings.Contains(s, `"retrieval":`) || strings.Contains(s, `"web_search":`) {
		t.Errorf("function tool leaked retrieval/web_search payload: %s", s)
	}
}

// NewRetrievalTool / NewWebSearchTool must emit their respective shapes. NOT
// VERIFIED LIVE — these pin the modeled shape, not a confirmed wire contract.
func TestRetrievalAndWebSearchToolMarshal(t *testing.T) {
	ret := NewRetrievalTool("kb-1", "find docs")
	out, err := json.Marshal(ret)
	if err != nil {
		t.Fatalf("marshal retrieval: %v", err)
	}
	s := string(out)
	for _, want := range []string{`"type":"retrieval"`, `"retrieval":{`, `"knowledge_id":"kb-1"`, `"prompt":"find docs"`} {
		if !strings.Contains(s, want) {
			t.Errorf("expected %q in %s", want, s)
		}
	}
	if strings.Contains(s, `"function":`) {
		t.Errorf("retrieval tool leaked function payload: %s", s)
	}

	ws := NewWebSearchTool("golang generics", "go modules")
	out, err = json.Marshal(ws)
	if err != nil {
		t.Fatalf("marshal web_search: %v", err)
	}
	s = string(out)
	for _, want := range []string{`"type":"web_search"`, `"web_search":{`, `"search_query":["golang generics","go modules"]`} {
		if !strings.Contains(s, want) {
			t.Errorf("expected %q in %s", want, s)
		}
	}
	// No speculative enable/search_result fields should leak.
	for _, unwanted := range []string{`"enable"`, `"search_result"`, `"enable_search"`} {
		if strings.Contains(s, unwanted) {
			t.Errorf("web_search tool leaked %q: %s", unwanted, s)
		}
	}
}

// Tool unmarshal must round-trip all three tool types by their Type field.
func TestToolUnmarshalRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		tool Tool
	}{
		{"function", NewFunctionTool("f", "d", map[string]any{"type": "object"})},
		{"retrieval", NewRetrievalTool("kb", "p")},
		{"web_search", NewWebSearchTool("q")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.tool)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got Tool
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Type != tc.tool.Type {
				t.Errorf("Type mismatch: got %q, want %q", got.Type, tc.tool.Type)
			}
			switch tc.name {
			case "function":
				if got.Function == nil || got.Function.Name != tc.tool.Function.Name {
					t.Errorf("Function not round-tripped: %+v vs %+v", got.Function, tc.tool.Function)
				}
			case "retrieval":
				if got.Retrieval == nil || got.Retrieval.KnowledgeID != tc.tool.Retrieval.KnowledgeID {
					t.Errorf("Retrieval not round-tripped: %+v vs %+v", got.Retrieval, tc.tool.Retrieval)
				}
			case "web_search":
				if got.WebSearch == nil || len(got.WebSearch.SearchQuery) != 1 {
					t.Errorf("WebSearch not round-tripped: %+v vs %+v", got.WebSearch, tc.tool.WebSearch)
				}
			}
		})
	}
}

// StreamToolCall must serialize to the stream_tool_call field when true, and
// be omitted when false (the default).
func TestStreamToolCallSerialization(t *testing.T) {
	with := ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}, TopP: 0.95, StreamToolCall: true}
	out, err := json.Marshal(with)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"stream_tool_call":true`) {
		t.Errorf("expected stream_tool_call:true in %s", out)
	}

	without := ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}, TopP: 0.95}
	out, err = json.Marshal(without)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(out), "stream_tool_call") {
		t.Errorf("stream_tool_call should be omitted when false: %s", out)
	}
}

// ChatResponse must deserialize a top-level web_search array into the
// WebSearch field (NOT VERIFIED LIVE — pins the modeled placement).
func TestChatResponseWebSearchUnmarshal(t *testing.T) {
	body := `{"id":"1","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"see sources"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2},"web_search":[{"title":"Go Docs","content":"overview","link":"https://go.dev/doc","media":"go.dev","icon":"go","refer":"1","publish_date":"2024-01-01"}]}`
	var resp ChatResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.WebSearch) != 1 {
		t.Fatalf("expected 1 web_search entry, got %d", len(resp.WebSearch))
	}
	w := resp.WebSearch[0]
	if w.Title != "Go Docs" || w.Link != "https://go.dev/doc" || w.Media != "go.dev" {
		t.Errorf("web_search entry mismatch: %+v", w)
	}
}

// A ChatResponse without a web_search array must leave WebSearch nil (not error).
func TestChatResponseWithoutWebSearch(t *testing.T) {
	body := `{"id":"1","model":"m","choices":[],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`
	var resp ChatResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.WebSearch != nil {
		t.Errorf("expected nil WebSearch, got %+v", resp.WebSearch)
	}
}

// The FinishReason* constants must match the values documented on docs.z.ai.
func TestFinishReasonConstants(t *testing.T) {
	want := map[string]string{
		"stop":                          FinishReasonStop,
		"tool_calls":                    FinishReasonToolCalls,
		"length":                        FinishReasonLength,
		"sensitive":                     FinishReasonSensitive,
		"model_context_window_exceeded": FinishReasonModelContextWindowExceeded,
		"network_error":                 FinishReasonNetworkError,
	}
	for value, constVal := range want {
		if value != constVal {
			t.Errorf("FinishReason constant mismatch: got %q, want %q", constVal, value)
		}
	}
}
