package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
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

// Create creates a chat completion
func (s *ChatService) Create(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if err := validateChatRequest(&req); err != nil {
		return nil, fmt.Errorf("invalid chat request: %w", err)
	}
	req.Tools = s.compatTools(req.Tools)

	var response ChatResponse
	err := s.client.doRequest(ctx, "POST", "/chat/completions", req, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat completion: %w", err)
	}

	return &response, nil
}

// CreateAsync submits a chat completion request and returns immediately
// with a task to poll via Client.GetAsyncResult/WaitForResult — useful for
// long-running generations where you don't want to hold a connection open.
// The request shape is identical to Create; only the endpoint and response
// differ (confirmed against docs.bigmodel.cn's live OpenAPI spec,
// POST /paas/v4/async/chat/completions -> AsyncResponse). Poll the
// returned ID with GetAsyncResult, whose AsyncResultResponse now also
// carries Choices/Usage for a completed chat task (see async.go).
func (s *ChatService) CreateAsync(ctx context.Context, req ChatRequest) (*AsyncTaskResponse, error) {
	if err := validateChatRequest(&req); err != nil {
		return nil, fmt.Errorf("invalid chat request: %w", err)
	}
	if req.Stream {
		return nil, fmt.Errorf("stream is not supported for async chat completions")
	}
	req.Tools = s.compatTools(req.Tools)

	var response AsyncTaskResponse
	if err := s.client.doRequest(ctx, "POST", "/async/chat/completions", req, &response); err != nil {
		return nil, fmt.Errorf("failed to submit async chat completion: %w", err)
	}
	return &response, nil
}

// CreateSimple creates a simple chat completion with basic parameters
func (s *ChatService) CreateSimple(ctx context.Context, model, userMessage string, messages []Message) (*ChatResponse, error) {
	if len(messages) == 0 {
		messages = []Message{
			{Role: "user", Content: userMessage},
		}
	}

	req := ChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: 0.7,
		TopP:        0.95, // Set default top_p value
		MaxTokens:   4096,
	}

	return s.Create(ctx, req)
}

// sseDone is the sentinel the server sends to terminate an SSE stream.
const sseDone = "[DONE]"

// CreateStream sends a streaming chat completion. onChunk is invoked once per
// SSE event (one delta at a time); returning a non-nil error aborts the stream.
// The request is sent with stream=true. Connect-level transient failures (429,
// 5xx, network errors) are retried up to Config.MaxRetries exactly like Create;
// once the stream has begun, mid-stream failures are surfaced, not retried.
func (s *ChatService) CreateStream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk) error) error {
	if err := validateChatRequest(&req); err != nil {
		return fmt.Errorf("invalid chat request: %w", err)
	}
	req.Stream = true
	req.Tools = s.compatTools(req.Tools)

	maxRetries := s.client.config.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		resp, err := s.client.send(ctx, s.client.config.BaseURL, s.client.config.APIKey, "POST", "/chat/completions", req)
		if err != nil {
			lastErr = fmt.Errorf("failed to execute request: %w", err)
			if attempt < maxRetries {
				s.client.backoff(ctx, "", attempt)
				continue
			}
			return lastErr
		}

		if resp.StatusCode != http.StatusOK {
			retryAfter := resp.Header.Get("Retry-After")
			apiErr := parseAPIError(resp)
			resp.Body.Close()

			retriable := false
			if ae, ok := apiErr.(*APIError); ok {
				retriable = ae.IsRetriable
			}
			lastErr = apiErr
			if attempt < maxRetries && retriable {
				s.client.backoff(ctx, retryAfter, attempt)
				continue
			}
			return apiErr
		}

		// Stream began — parse SSE and deliver chunks. No retry past this point.
		err = s.readSSE(ctx, resp, onChunk)
		resp.Body.Close()
		return err
	}
	return lastErr
}

// readSSE parses a Server-Sent-Events stream, invoking onChunk for each
// `data:` payload and returning on `[DONE]` or stream end.
func (s *ChatService) readSSE(ctx context.Context, resp *http.Response, onChunk func(StreamChunk) error) error {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := scanner.Text()
		if len(line) == 0 || line[0] == ':' {
			continue // event separator or SSE comment / keep-alive
		}
		const prefix = "data:"
		if !strings.HasPrefix(line, prefix) {
			continue // ignore event:, id:, retry: control lines
		}
		payload := strings.TrimSpace(line[len(prefix):])
		if payload == "" {
			continue
		}
		if payload == sseDone {
			return nil
		}
		var chunk StreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return fmt.Errorf("failed to parse stream chunk: %w", err)
		}
		if err := onChunk(chunk); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream read error: %w", err)
	}
	return nil
}

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

	// Validate top_p range
	if req.TopP < 0.01 || req.TopP > 1 {
		return fmt.Errorf("top_p must be between 0.01 and 1")
	}

	// Validate Thinking.Effort (when set) against the documented enum so the
	// caller gets a clear error rather than the server's 400.
	if req.Thinking != nil && req.Thinking.Effort != "" {
		if !validEffort[req.Thinking.Effort] {
			return fmt.Errorf("thinking.effort %q is not one of %v", req.Thinking.Effort, effortValues())
		}
	}

	// Enforce the documented 128-function cap and the tool-name pattern
	// (^[A-Za-z0-9_-]{1,64}$) client-side. Each tool type must also carry its
	// matching payload (Function/Retrieval/WebSearch) — a bare {"type":...}
	// would otherwise serialize and likely 400 at the server.
	funcCount := 0
	for i, t := range req.Tools {
		switch t.Type {
		case "", ToolTypeFunction:
			if t.Function == nil {
				return fmt.Errorf("tools[%d]: type %q requires a function payload", i, t.Type)
			}
			funcCount++
			if !toolNamePattern.MatchString(t.Function.Name) {
				return fmt.Errorf("tools[%d].function.name %q must match ^[A-Za-z0-9_-]{1,64}$", i, t.Function.Name)
			}
		case ToolTypeRetrieval:
			if t.Retrieval == nil {
				return fmt.Errorf("tools[%d]: type %q requires a retrieval payload", i, t.Type)
			}
		case ToolTypeWebSearch:
			if t.WebSearch == nil {
				return fmt.Errorf("tools[%d]: type %q requires a web_search payload", i, t.Type)
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

// validEffort is the set of ThinkingConfig.Effort values docs.z.ai documents.
var validEffort = map[string]bool{
	EffortMax: true, EffortXhigh: true, EffortHigh: true, EffortMedium: true,
	EffortLow: true, EffortMinimal: true, EffortNone: true,
}

func effortValues() []string {
	return []string{
		EffortMax, EffortXhigh, EffortHigh, EffortMedium,
		EffortLow, EffortMinimal, EffortNone,
	}
}
