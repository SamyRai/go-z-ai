package cli

import (
	"testing"
)

// resetChatVars restores the sticky package-level chat flag vars after a test
// mutates them, so buildChatRequest cases don't leak into each other.
func resetChatVars(t *testing.T) {
	t.Helper()
	prev := struct {
		model, system, thinking, effort, schemaFile, toolFile string
		images                                                []string
	}{chatModel, chatSystemMsg, chatThinking, chatEffort, chatSchemaFile, chatToolFile, chatImages}
	t.Cleanup(func() {
		chatModel, chatSystemMsg = prev.model, prev.system
		chatThinking, chatEffort = prev.thinking, prev.effort
		chatSchemaFile, chatToolFile = prev.schemaFile, prev.toolFile
		chatImages = prev.images
	})
	// Start from a clean baseline.
	chatModel, chatSystemMsg = "glm-5.2", "sys"
	chatThinking, chatEffort = "", ""
	chatSchemaFile, chatToolFile = "", ""
	chatImages = nil
}

func TestBuildChatRequestBase(t *testing.T) {
	resetChatVars(t)

	req, err := buildChatRequest("hello")
	if err != nil {
		t.Fatalf("buildChatRequest: %v", err)
	}
	if len(req.Messages) != 2 || req.Messages[0].Role != "system" || req.Messages[1].Content != "hello" {
		t.Errorf("unexpected messages: %+v", req.Messages)
	}
	if req.Thinking != nil {
		t.Errorf("expected no thinking config by default, got %+v", req.Thinking)
	}
}

func TestBuildChatRequestThinking(t *testing.T) {
	resetChatVars(t)
	chatThinking, chatEffort = "enabled", "high"

	req, err := buildChatRequest("hi")
	if err != nil {
		t.Fatalf("buildChatRequest: %v", err)
	}
	if req.Thinking == nil || req.Thinking.Type != "enabled" || req.Thinking.Effort != "high" {
		t.Errorf("expected thinking enabled/high, got %+v", req.Thinking)
	}
}

func TestBuildChatRequestImages(t *testing.T) {
	resetChatVars(t)
	chatImages = []string{"https://example.com/a.png"}

	req, err := buildChatRequest("describe")
	if err != nil {
		t.Fatalf("buildChatRequest: %v", err)
	}
	last := req.Messages[len(req.Messages)-1]
	if len(last.Images) != 1 || last.Images[0] != "https://example.com/a.png" {
		t.Errorf("expected image attached to user message, got %+v", last.Images)
	}
}

func TestBuildChatRequestBadImageFails(t *testing.T) {
	resetChatVars(t)
	chatImages = []string{"@/nonexistent/path/x.png"}

	if _, err := buildChatRequest("x"); err == nil {
		t.Fatal("expected an error for a missing --image file")
	}
}
