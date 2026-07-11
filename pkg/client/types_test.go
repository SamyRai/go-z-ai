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
