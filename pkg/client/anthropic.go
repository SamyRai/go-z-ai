package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// AnthropicService talks to Z.AI's Anthropic-compatible Messages API at
// AnthropicBaseURL (https://api.z.ai/api/anthropic), the same surface the
// GLM Coding Plan points Claude Code and other Anthropic-protocol tools at via
// pkg/coding. It is a typed Go client for POST /v1/messages, parallel to
// ChatService's OpenAI-style /chat/completions.
//
// Authentication is the z.ai API key as a Bearer token (Config.APIKey), not
// Anthropic's x-api-key header — this matches how @z_ai/coding-helper wires
// ANTHROPIC_AUTH_TOKEN for Claude Code, which is the configuration Z.AI's
// endpoint is known to accept. An anthropic-version header is sent on every
// request.
//
// NOT VERIFIED LIVE: the request/response shapes below follow Anthropic's
// documented Messages API (which Z.AI mirrors) but have not been confirmed
// against a real successful call from the development account (no Coding Plan
// entitlement on it). If you can record one and hit a shape mismatch, please
// open an issue or PR with a cassette — see docs/roadmap.md.
type AnthropicService struct {
	client *Client
}

// AnthropicVersion is sent as the anthropic-version header, matching the value
// Claude Code uses against this endpoint.
const AnthropicVersion = "2023-06-01"

// anthropicMessagesEndpoint is the Messages path under AnthropicBaseURL.
const anthropicMessagesEndpoint = "/v1/messages"

// AnthropicMessageRequest is a POST /v1/messages request body. MaxTokens is
// required by the Messages API. Temperature/TopP/TopK are pointers so an unset
// value is omitted rather than sent as a (meaningful) zero.
type AnthropicMessageRequest struct {
	Model         string               `json:"model"`
	Messages      []AnthropicMessage   `json:"messages"`
	MaxTokens     int                  `json:"max_tokens"`
	System        string               `json:"system,omitempty"`
	Temperature   *float64             `json:"temperature,omitempty"`
	TopP          *float64             `json:"top_p,omitempty"`
	TopK          *int                 `json:"top_k,omitempty"`
	StopSequences []string             `json:"stop_sequences,omitempty"`
	Stream        bool                 `json:"stream,omitempty"`
	Tools         []AnthropicTool      `json:"tools,omitempty"`
	ToolChoice    *AnthropicToolChoice `json:"tool_choice,omitempty"`
	Thinking      *AnthropicThinking   `json:"thinking,omitempty"`
	Metadata      map[string]any       `json:"metadata,omitempty"`
}

// AnthropicThinking enables Anthropic extended thinking. When Type is
// "enabled", BudgetTokens caps the reasoning token budget (and must be less
// than MaxTokens). GLM models are reasoning models, so this is the typed knob
// for surfacing their chain-of-thought as thinking content blocks.
type AnthropicThinking struct {
	Type         string `json:"type"`                    // enabled, disabled
	BudgetTokens int    `json:"budget_tokens,omitempty"` // required when enabled
}

// AnthropicMessage is one turn. Content is always sent as a list of blocks
// (the Messages API accepts blocks for every message), avoiding the
// string-or-array union — use AnthropicTextMessage for the common text case.
type AnthropicMessage struct {
	Role    string                  `json:"role"` // user, assistant
	Content []AnthropicContentBlock `json:"content"`
}

// AnthropicContentBlock is a single content block. Only the fields relevant to
// Type are populated; the rest are omitted. It covers the block types a chat
// exchange uses: text, image, tool_use (assistant), and tool_result (user).
type AnthropicContentBlock struct {
	Type string `json:"type"` // text, image, tool_use, tool_result

	// type: text
	Text string `json:"text,omitempty"`

	// type: thinking (extended reasoning). A thinking-enabled request may
	// return these blocks before the answer; redacted_thinking blocks carry
	// their payload in Data instead.
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`
	Data      string `json:"data,omitempty"` // type: redacted_thinking

	// type: image
	Source *AnthropicImageSource `json:"source,omitempty"`

	// type: tool_use
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// type: tool_result
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"` // string or nested blocks
	IsError   bool            `json:"is_error,omitempty"`
}

// AnthropicImageSource is the source of an image content block (base64 or url).
type AnthropicImageSource struct {
	Type      string `json:"type"`                 // base64, url
	MediaType string `json:"media_type,omitempty"` // e.g. image/png (base64 only)
	Data      string `json:"data,omitempty"`       // base64-encoded bytes
	URL       string `json:"url,omitempty"`        // when Type == "url"
}

// AnthropicTool declares a callable tool. InputSchema is a JSON Schema for the
// tool's arguments and is subject to the same GLM schema-compatibility rewrite
// as ChatService tools (see SanitizeToolSchemas / Config.DisableToolSchemaCompat).
type AnthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema,omitempty"`
}

// AnthropicToolChoice controls whether/which tool the model must call.
type AnthropicToolChoice struct {
	Type string `json:"type"`           // auto, any, tool
	Name string `json:"name,omitempty"` // required when Type == "tool"
}

// AnthropicResponse is a completed (non-streaming) Messages response.
type AnthropicResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"` // "message"
	Role         string                  `json:"role"` // "assistant"
	Model        string                  `json:"model"`
	Content      []AnthropicContentBlock `json:"content"`
	StopReason   string                  `json:"stop_reason"` // end_turn, max_tokens, stop_sequence, tool_use
	StopSequence *string                 `json:"stop_sequence"`
	Usage        AnthropicUsage          `json:"usage"`

	// ReasoningContent is not part of Anthropic's Messages schema, but GLM's
	// endpoint may surface reasoning in this OpenAI-style field instead of a
	// thinking content block (the reasoning_content-not-converted case behind
	// claude-code-router#1133). Thinking() falls back to it. See docs/roadmap.md.
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

// AnthropicUsage is the token accounting on a Messages response.
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicTextMessage builds a message whose content is a single text block —
// the common case for plain user/assistant turns.
func AnthropicTextMessage(role, text string) AnthropicMessage {
	return AnthropicMessage{
		Role:    role,
		Content: []AnthropicContentBlock{{Type: "text", Text: text}},
	}
}

// Text returns the concatenated text of every text block in the response, the
// usual thing a caller wants from a non-tool answer.
func (r *AnthropicResponse) Text() string {
	var b strings.Builder
	for _, block := range r.Content {
		if block.Type == "text" {
			b.WriteString(block.Text)
		}
	}
	return b.String()
}

// Thinking returns the model's reasoning: the concatenated text of every
// thinking block, or — when the endpoint surfaced reasoning in the OpenAI-style
// reasoning_content field instead of a thinking block — that field. Empty when
// the request didn't enable thinking.
func (r *AnthropicResponse) Thinking() string {
	var b strings.Builder
	for _, block := range r.Content {
		if block.Type == "thinking" {
			b.WriteString(block.Thinking)
		}
	}
	if b.Len() == 0 {
		return r.ReasoningContent
	}
	return b.String()
}

// Create sends a POST /v1/messages request and returns the completed message.
func (s *AnthropicService) Create(ctx context.Context, req AnthropicMessageRequest) (*AnthropicResponse, error) {
	if err := validateAnthropicRequest(&req); err != nil {
		return nil, fmt.Errorf("invalid anthropic request: %w", err)
	}
	req.Tools = s.compatTools(req.Tools)

	var resp AnthropicResponse
	if err := s.client.doRequestBaseKeyHeaders(ctx, AnthropicBaseURL, s.client.config.APIKey, "POST", anthropicMessagesEndpoint, req, &resp, anthropicHeaders()); err != nil {
		return nil, fmt.Errorf("failed to create anthropic message: %w", err)
	}
	return &resp, nil
}

// AnthropicStreamEvent is one Server-Sent Event from a streaming Messages
// response. Type is the SSE event name (message_start, content_block_start,
// content_block_delta, content_block_stop, message_delta, message_stop, ping,
// error); Data is the raw JSON payload, which the caller unmarshals according
// to Type. The raw form is deliberate: it faithfully passes through Anthropic's
// event protocol without this client having to model every delta subtype.
type AnthropicStreamEvent struct {
	Type string
	Data json.RawMessage
}

// CreateStream sends a streaming POST /v1/messages request, invoking onEvent
// once per SSE event. Returning a non-nil error from onEvent aborts the stream.
// Connect-level transient failures (429/5xx/network) are retried like Create;
// once the stream has begun, mid-stream failures are surfaced, not retried.
func (s *AnthropicService) CreateStream(ctx context.Context, req AnthropicMessageRequest, onEvent func(AnthropicStreamEvent) error) error {
	if err := validateAnthropicRequest(&req); err != nil {
		return fmt.Errorf("invalid anthropic request: %w", err)
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

		resp, err := s.client.sendHeaders(ctx, AnthropicBaseURL, s.client.config.APIKey, "POST", anthropicMessagesEndpoint, req, anthropicHeaders())
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

		err = readAnthropicSSE(ctx, resp, onEvent)
		resp.Body.Close()
		return err
	}
	return lastErr
}

// readAnthropicSSE parses Anthropic's `event:`/`data:` SSE framing, pairing
// each event name with its data payload and delivering one AnthropicStreamEvent
// per event.
func readAnthropicSSE(ctx context.Context, resp *http.Response, onEvent func(AnthropicStreamEvent) error) error {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var eventType string
	var data strings.Builder

	flush := func() error {
		if data.Len() == 0 && eventType == "" {
			return nil
		}
		payload := strings.TrimSpace(data.String())
		ev := AnthropicStreamEvent{Type: eventType}
		if payload != "" {
			ev.Data = json.RawMessage(payload)
		}
		eventType = ""
		data.Reset()
		if payload == "" && ev.Type == "" {
			return nil
		}
		return onEvent(ev)
	}

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := scanner.Text()
		if line == "" {
			// Blank line terminates one event.
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if line[0] == ':' {
			continue // SSE comment / keep-alive
		}
		if name, ok := strings.CutPrefix(line, "event:"); ok {
			eventType = strings.TrimSpace(name)
			continue
		}
		if payload, ok := strings.CutPrefix(line, "data:"); ok {
			// SSE joins multiple data: lines within one event with newlines.
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimSpace(payload))
			continue
		}
		// Ignore id:/retry: and any other control lines.
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream read error: %w", err)
	}
	// Deliver a trailing event with no terminating blank line.
	return flush()
}

// compatTools applies the GLM tool-schema compatibility rewrite to each tool's
// InputSchema, unless the caller opted out via Config.DisableToolSchemaCompat.
// It never mutates the caller's slice.
func (s *AnthropicService) compatTools(tools []AnthropicTool) []AnthropicTool {
	if s.client.config.DisableToolSchemaCompat || len(tools) == 0 {
		return tools
	}
	out := make([]AnthropicTool, len(tools))
	for i, t := range tools {
		out[i] = t
		if len(t.InputSchema) > 0 {
			out[i].InputSchema = sanitizeParameters(t.InputSchema)
		}
	}
	return out
}

func anthropicHeaders() map[string]string {
	return map[string]string{"anthropic-version": AnthropicVersion}
}

func validateAnthropicRequest(req *AnthropicMessageRequest) error {
	if req.Model == "" {
		return fmt.Errorf("model is required")
	}
	if len(req.Messages) == 0 {
		return fmt.Errorf("at least one message is required")
	}
	if req.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens is required and must be positive")
	}
	hasUser := false
	for _, m := range req.Messages {
		if m.Role == "user" {
			hasUser = true
			break
		}
	}
	if !hasUser {
		return fmt.Errorf("messages must contain at least one user message")
	}
	return nil
}
