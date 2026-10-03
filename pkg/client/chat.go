package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"regexp"
	"slices"
	"strings"
)

// ChatService handles chat completion operations
type ChatService struct {
	client *Client
}

// compatTools normalizes tool parameter schemas for GLM's strict parser unless
// the caller opted out via Config.DisableToolSchemaCompat. It is a no-op for
// requests without tools or with already-flat schemas, and never mutates the
// caller's tool slice (SanitizeToolSchemas returns fresh copies).
func (s *ChatService) compatTools(tools []Tool) []Tool {
	if s.client.config.DisableToolSchemaCompat {
		return tools
	}
	return SanitizeToolSchemas(tools)
}

// Create creates a chat completion.
func (s *ChatService) Create(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	r, err := s.prepare(req, "/chat/completions")
	if err != nil {
		return nil, err
	}
	var response ChatResponse
	if err := s.client.do(ctx, r, &response); err != nil {
		return nil, fmt.Errorf("failed to create chat completion: %w", err)
	}
	return &response, nil
}

// CreateAsync submits a chat completion request and returns immediately with
// a task to poll via Client.GetAsyncResult/WaitForResult — useful for
// long-running generations where you don't want to hold a connection open.
// The request shape is identical to Create; only the endpoint and response
// differ (POST /async/chat/completions -> AsyncTaskResponse). A completed
// task's AsyncResultResponse carries Choices/Usage (see async.go).
func (s *ChatService) CreateAsync(ctx context.Context, req ChatRequest) (*AsyncTaskResponse, error) {
	if req.Stream {
		return nil, fmt.Errorf("stream is not supported for async chat completions")
	}
	r, err := s.prepare(req, "/async/chat/completions")
	if err != nil {
		return nil, err
	}
	var response AsyncTaskResponse
	if err := s.client.do(ctx, r, &response); err != nil {
		return nil, fmt.Errorf("failed to submit async chat completion: %w", err)
	}
	return &response, nil
}

// Stream sends a streaming chat completion and returns an iterator over its
// chunks (Go 1.23+ range-over-func):
//
//	for chunk, err := range c.Chat().Stream(ctx, req) {
//	    if err != nil { /* terminal — the loop ends */ break }
//	    if len(chunk.Choices) > 0 {
//	        fmt.Print(chunk.Choices[0].Delta.Content)
//	    }
//	}
//
// The request is sent with stream=true. Connect-level transient failures are
// retried like Create; a mid-stream failure is the iterator's terminal error.
// Breaking out of the loop or cancelling ctx stops the stream and releases
// the connection.
func (s *ChatService) Stream(ctx context.Context, req ChatRequest) iter.Seq2[StreamChunk, error] {
	req.Stream = true
	r, err := s.prepare(req, "/chat/completions")
	if err != nil {
		return errorIter[StreamChunk](err)
	}
	return streamSSE(ctx, s.client, r, decodeChatStream)
}

// prepare validates req, applies the tool-schema compatibility rewrite, and
// wraps it as an apiRequest for path.
func (s *ChatService) prepare(req ChatRequest, path string) (apiRequest, error) {
	if err := validateChatRequest(&req); err != nil {
		return apiRequest{}, fmt.Errorf("invalid chat request: %w", err)
	}
	req.Tools = s.compatTools(req.Tools)
	return apiRequest{method: "POST", path: path, body: req, service: "chat", model: req.Model}, nil
}

// sseDone is the sentinel data payload that terminates an OpenAI-style stream.
const sseDone = "[DONE]"

// decodeChatStream decodes an OpenAI-style chat SSE body — one JSON
// StreamChunk per data payload, terminated by [DONE] or end of stream.
func decodeChatStream(ctx context.Context, body io.Reader, emit func(StreamChunk) error) error {
	err := scanSSE(ctx, body, func(_, data string) error {
		switch data {
		case "":
			return nil
		case sseDone:
			return errStreamDone
		}
		if strings.Contains(data, `"error"`) {
			if err := streamError([]byte(data)); err != nil {
				return err
			}
		}
		var chunk StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("failed to parse stream chunk: %w", err)
		}
		return emit(chunk)
	})
	if errors.Is(err, errStreamDone) {
		return nil
	}
	return err
}

// errStreamDone ends scanning at the [DONE] sentinel; never surfaced.
var errStreamDone = errors.New("stream done")

// toolNamePattern matches Z.AI's documented constraint on tools[].function.name:
// ASCII letters, digits, underscore, hyphen; 1–64 chars. Applied in
// validateChatRequest so callers get a clear client-side error instead of an
// opaque server-side one. NOT VERIFIED LIVE that the server rejects names
// outside this set — the regex mirrors docs.z.ai's stated rule.
var toolNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func validateChatRequest(req *ChatRequest) error {
	if req.Model == "" {
		return fmt.Errorf("model is required")
	}
	if len(req.Messages) == 0 {
		return fmt.Errorf("at least one message is required")
	}

	// Validate messages contain at least one user message
	hasUserMessage := false
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			hasUserMessage = true
			break
		}
	}
	if !hasUserMessage {
		return fmt.Errorf("messages must contain at least one user message")
	}

	// Validate temperature range
	if req.Temperature < 0 || req.Temperature > 1 {
		return fmt.Errorf("temperature must be between 0 and 1")
	}

	// Validate top_p range. TopP == 0 means "unset" (omitempty drops it on the
	// wire → server default); allow it like Temperature rather than rejecting
	// the natural zero-value struct literal.
	if req.TopP < 0 || req.TopP > 1 {
		return fmt.Errorf("top_p must be between 0 and 1")
	}

	if err := validateEffort(req.Model, req.ReasoningEffort); err != nil {
		return err
	}

	// Enforce the documented 128-function cap and the tool-name pattern
	// client-side. Each tool type must carry its matching payload — a bare
	// {"type":...} would serialize and be rejected by the server.
	funcCount := 0
	for i, t := range req.Tools {
		switch t.Type {
		case "", ToolTypeFunction:
			if t.Function == nil {
				return fmt.Errorf("tools[%d]: type %q requires a function payload", i, t.Type)
			}
			funcCount++
			if !toolNamePattern.MatchString(t.Function.Name) {
				return fmt.Errorf("tools[%d].function.name %q must match %s", i, t.Function.Name, toolNamePattern)
			}
		case ToolTypeWebSearch:
			if t.WebSearch == nil {
				return fmt.Errorf("tools[%d]: type %q requires a web_search payload", i, t.Type)
			}
		case ToolTypeRetrieval:
			if t.Retrieval == nil || t.Retrieval.KnowledgeID == "" {
				return fmt.Errorf("tools[%d]: type %q requires a retrieval payload with a knowledge_id", i, t.Type)
			}
		case ToolTypeMCP:
			if t.MCP == nil || t.MCP.ServerLabel == "" {
				return fmt.Errorf("tools[%d]: type %q requires an mcp payload with a server_label", i, t.Type)
			}
		default:
			return fmt.Errorf("tools[%d]: unknown tool type %q", i, t.Type)
		}
	}
	if funcCount > ToolMaxFunctions {
		return fmt.Errorf("too many function tools: %d (max %d)", funcCount, ToolMaxFunctions)
	}

	return nil
}

// validateEffort checks a reasoning_effort value against the levels model
// accepts: its catalog entry's list when the model is cataloged (an empty
// list means the model does not take the parameter), otherwise every
// documented level.
func validateEffort(model, effort string) error {
	if effort == "" {
		return nil
	}
	allowed := AllEfforts
	if entry := findCatalogEntry(model); entry != nil {
		allowed = entry.ReasoningEfforts
	}
	switch {
	case len(allowed) == 0:
		return fmt.Errorf("%s does not accept reasoning_effort", model)
	case !slices.Contains(allowed, effort):
		return fmt.Errorf("reasoning_effort %q is not supported by %s (allowed: %s)", effort, model, strings.Join(allowed, ", "))
	}
	return nil
}
