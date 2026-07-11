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
