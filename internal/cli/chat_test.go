package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

func TestChatRequestBase(t *testing.T) {
	req, err := chatOptions{model: client.DefaultModel}.request("hello")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if len(req.Messages) != 1 || req.Messages[0].Content != "hello" {
		t.Errorf("unexpected messages: %+v", req.Messages)
	}
	if req.Thinking != nil || req.DoSample != nil || req.Temperature != 0 || req.MaxTokens != 0 {
		t.Errorf("unset flags must leave the model defaults alone: %+v", req)
	}
}

func TestChatRequestReasoning(t *testing.T) {
	req, err := chatOptions{model: client.DefaultModel, thinking: client.ThinkingEnabled, effort: client.EffortHigh}.request("hi")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if req.Thinking == nil || req.Thinking.Type != "enabled" || req.ReasoningEffort != "high" {
		t.Errorf("thinking/effort not applied: %+v %q", req.Thinking, req.ReasoningEffort)
	}
}

// --json-schema switches to JSON-object mode and describes the schema in the
// system message (the API has no json_schema response format).
func TestChatRequestJSONSchema(t *testing.T) {
	req, err := chatOptions{system: "be terse", schema: `{"type":"object"}`}.request("hi")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if req.ResponseFormat == nil || req.ResponseFormat.Type != client.ResponseFormatJSONObject {
		t.Errorf("response format = %+v", req.ResponseFormat)
	}
	sys := req.Messages[0]
	if sys.Role != "system" || !strings.Contains(sys.Content, "be terse") || !strings.Contains(sys.Content, `{"type":"object"}`) {
		t.Errorf("system message = %+v", sys)
	}
	if _, err := (chatOptions{schema: "{nope"}).request("hi"); err == nil {
		t.Error("invalid schema JSON must fail")
	}
}

// Attachments resolve URLs and @files into the user message.
func TestChatRequestAttachments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clip.mp4")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	req, err := chatOptions{images: []string{"https://x/a.png"}, videos: []string{"@" + path}}.request("describe")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	user := req.Messages[len(req.Messages)-1]
	if len(user.Images) != 1 || len(user.Videos) != 1 || !strings.HasPrefix(user.Videos[0], "data:video/mp4;base64,") {
		t.Errorf("attachments = %+v / %+v", user.Images, user.Videos)
	}
	if _, err := (chatOptions{images: []string{"@/nonexistent/x.png"}}).request("x"); err == nil {
		t.Error("a missing attachment must fail")
	}
}
