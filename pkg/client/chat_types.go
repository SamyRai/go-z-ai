package client

import (
	"encoding/json"
	"errors"
)

// Wire types for POST /chat/completions (and its async and streaming forms).
// Field names follow the official Z.AI SDKs (zai-sdk 0.2.3 / Java 0.3.5) and
// docs.z.ai's chat-completion reference, checked 2026-10-02.

// Message is one chat turn. Content is plain text for ordinary use; set
// Images, Videos, or Files (https:// URLs or data: URIs) to attach media for
// multimodal models — MarshalJSON switches Content to the content-parts wire
// shape transparently (see content.go).
type Message struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"-"` // image_url parts
	Videos  []string `json:"-"` // video_url parts
	Files   []string `json:"-"` // file_url parts
	// ReasoningContent carries an assistant turn's reasoning back to the
	// model. Echo it unchanged on assistant messages when using preserved
	// thinking (ThinkingConfig.ClearThinking = false) or interleaved
	// thinking with tools; RunWithTools does this automatically.
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`   // assistant messages that request tool calls
	ToolCallID       string     `json:"tool_call_id,omitempty"` // role "tool": the call being answered
	Name             string     `json:"name,omitempty"`
}

// ChatRequest is a chat completion request.
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	// RequestID is a caller-chosen unique ID for the request; the platform
	// generates one when empty.
	RequestID string `json:"request_id,omitempty"`
	// UserID identifies the end user (6–128 chars) for abuse monitoring.
	UserID string `json:"user_id,omitempty"`
	// DoSample enables sampling (API default true). When false, temperature
	// and top_p are ignored. A pointer so an explicit false is sent.
	DoSample    *bool    `json:"do_sample,omitempty"`
	Temperature float64  `json:"temperature,omitempty"` // [0, 1]; 0 = server default
	TopP        float64  `json:"top_p,omitempty"`       // [0, 1]; 0 = server default
	MaxTokens   int      `json:"max_tokens,omitempty"`
	Stop        []string `json:"stop,omitempty"` // the API honors one stop word
	Stream      bool     `json:"stream,omitempty"`
	// ToolStream streams tool-call arguments incrementally (StreamDelta.
	// ToolCalls, reassembled by ToolCall.Index) instead of in one final
	// chunk. Supported by GLM-4.6 and later.
	ToolStream     bool            `json:"tool_stream,omitempty"`
	Tools          []Tool          `json:"tools,omitempty"`
	ToolChoice     string          `json:"tool_choice,omitempty"` // the API supports only "auto"
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
	Thinking       *ThinkingConfig `json:"thinking,omitempty"`
	// ReasoningEffort sets how hard a thinking model reasons (one of the
	// Effort* constants; API default max). GLM-5.2 accepts every level;
	// GLM-5.3-family models accept only low, high, and max. Validated against
	// the model's catalog entry when the model is known.
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
}

// ThinkingConfig.Type values.
const (
	ThinkingEnabled  = "enabled"
	ThinkingDisabled = "disabled"
)

// ThinkingConfig controls chain-of-thought reasoning (GLM-4.5 and later).
// GLM-5.3-family models always think; disabling is not supported there.
type ThinkingConfig struct {
	Type string `json:"type,omitempty"` // ThinkingEnabled or ThinkingDisabled
	// ClearThinking controls whether reasoning_content from earlier turns is
	// dropped (API default true). Set it to false for preserved thinking —
	// and echo each assistant turn's ReasoningContent back unchanged.
	ClearThinking *bool `json:"clear_thinking,omitempty"`
}

// ReasoningEffort levels. The server normalizes none/minimal to "skip
// thinking", low/medium to high, and xhigh to max.
const (
	EffortMax     = "max"
	EffortXhigh   = "xhigh"
	EffortHigh    = "high"
	EffortMedium  = "medium"
	EffortLow     = "low"
	EffortMinimal = "minimal"
	EffortNone    = "none"
)

// AllEfforts lists every documented ReasoningEffort level.
var AllEfforts = []string{EffortMax, EffortXhigh, EffortHigh, EffortMedium, EffortLow, EffortMinimal, EffortNone}

// ResponseFormat.Type values. The API supports plain text and JSON-object
// output; for structured output, use JSON-object mode and describe the
// schema in the system prompt (see docs.z.ai, "Structured Output").
const (
	ResponseFormatText       = "text"
	ResponseFormatJSONObject = "json_object"
)

// ResponseFormat controls the output format.
type ResponseFormat struct {
	Type string `json:"type"` // ResponseFormatText or ResponseFormatJSONObject
}

// JSONObjectFormat requests JSON-object output.
func JSONObjectFormat() *ResponseFormat { return &ResponseFormat{Type: ResponseFormatJSONObject} }

// JSONSchemaPrompt returns the system-prompt instruction that, together with
// JSONObjectFormat, asks for a JSON object conforming to schema — the
// documented way to get structured output, as the API has no json_schema
// response format.
func JSONSchemaPrompt(schema json.RawMessage) (string, error) {
	if !json.Valid(schema) {
		return "", errors.New("schema is not valid JSON")
	}
	return "Respond only with a JSON object that conforms to this JSON Schema:\n" + string(schema), nil
}

// Tool type identifiers for Tool.Type.
const (
	ToolTypeFunction  = "function"
	ToolTypeWebSearch = "web_search"
	ToolTypeRetrieval = "retrieval"
	ToolTypeMCP       = "mcp"
)

// ToolMaxFunctions is the documented cap on function tools in one request,
// enforced client-side by validateChatRequest.
const ToolMaxFunctions = 128

// Tool is a tool the model may use. Type selects which payload the server
// reads; exactly that payload must be set (see the New*Tool constructors).
type Tool struct {
	Type      string        `json:"type"`
	Function  *FunctionDef  `json:"function,omitempty"`
	WebSearch *WebSearchDef `json:"web_search,omitempty"`
	Retrieval *Retrieval    `json:"retrieval,omitempty"`
	MCP       *MCPServer    `json:"mcp,omitempty"`
}

// FunctionDef declares a callable function. Name must match
// ^[A-Za-z0-9_-]{1,64}$ (enforced by validateChatRequest).
type FunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// WebSearchDef configures the built-in web_search tool. Results the model
// grounded its answer in come back in ChatResponse.WebSearch when
// SearchResult is set.
type WebSearchDef struct {
	Enable       bool   `json:"enable"`
	SearchEngine string `json:"search_engine,omitempty"` // e.g. SearchEnginePrime
	// SearchQuery forces a specific query instead of letting the model
	// derive one from the conversation.
	SearchQuery  string `json:"search_query,omitempty"`
	SearchResult bool   `json:"search_result,omitempty"` // return sources in ChatResponse.WebSearch
	// SearchPrompt customizes how results are fed to the model; it may use
	// the {{search_result}} placeholder.
	SearchPrompt        string `json:"search_prompt,omitempty"`
	Count               int    `json:"count,omitempty"` // 1–50
	SearchDomainFilter  string `json:"search_domain_filter,omitempty"`
	SearchRecencyFilter string `json:"search_recency_filter,omitempty"` // SearchRecency* constants
	ContentSize         string `json:"content_size,omitempty"`          // SearchContent* constants
}

// Retrieval is the payload of a retrieval (knowledge-base) tool. The
// knowledge base product is served by the China platform.
type Retrieval struct {
	KnowledgeID string `json:"knowledge_id"`
	// PromptTemplate may use the {{knowledge}} and {{question}} placeholders.
	PromptTemplate string `json:"prompt_template,omitempty"`
}

// MCP transport types for MCPServer.TransportType.
const (
	MCPTransportSSE            = "sse"
	MCPTransportStreamableHTTP = "streamable-http" // API default
)

// MCPServer lets the model call tools on a remote MCP server server-side.
// ServerURL may be empty to use one of Z.AI's hosted MCP servers by
// ServerLabel.
type MCPServer struct {
	ServerLabel   string            `json:"server_label"`
	ServerURL     string            `json:"server_url,omitempty"`
	TransportType string            `json:"transport_type,omitempty"`
	AllowedTools  []string          `json:"allowed_tools,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
}

// NewFunctionTool builds a function tool.
func NewFunctionTool(name, description string, parameters map[string]any) Tool {
	return Tool{Type: ToolTypeFunction, Function: &FunctionDef{Name: name, Description: description, Parameters: parameters}}
}

// NewWebSearchTool builds an enabled web_search tool that returns its sources
// (ChatResponse.WebSearch) using engine (e.g. SearchEnginePrime).
func NewWebSearchTool(engine string) Tool {
	return Tool{Type: ToolTypeWebSearch, WebSearch: &WebSearchDef{Enable: true, SearchEngine: engine, SearchResult: true}}
}

// NewRetrievalTool builds a knowledge-base retrieval tool.
func NewRetrievalTool(knowledgeID, promptTemplate string) Tool {
	return Tool{Type: ToolTypeRetrieval, Retrieval: &Retrieval{KnowledgeID: knowledgeID, PromptTemplate: promptTemplate}}
}

// NewMCPTool builds an MCP tool for the server labeled label at url (empty
// for a Z.AI-hosted server), restricted to allowedTools when non-empty.
func NewMCPTool(label, url string, allowedTools ...string) Tool {
	return Tool{Type: ToolTypeMCP, MCP: &MCPServer{ServerLabel: label, ServerURL: url, AllowedTools: allowedTools}}
}

// ChatResponse is a chat completion response. WebSearch carries the sources
// a web_search tool grounded the answer in.
type ChatResponse struct {
	ID        string            `json:"id"`
	RequestID string            `json:"request_id,omitempty"`
	Created   int64             `json:"created"`
	Model     string            `json:"model"`
	Choices   []Choice          `json:"choices"`
	Usage     Usage             `json:"usage"`
	WebSearch []WebSearchResult `json:"web_search,omitempty"`
}

// GetUsage implements usageBearer for the observability hooks.
func (r *ChatResponse) GetUsage() *Usage { return &r.Usage }

// Choice is one response choice.
type Choice struct {
	Index        int         `json:"index"`
	Message      ResponseMsg `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// FinishReason values.
const (
	FinishReasonStop                       = "stop"
	FinishReasonToolCalls                  = "tool_calls"
	FinishReasonLength                     = "length"
	FinishReasonSensitive                  = "sensitive"
	FinishReasonModelContextWindowExceeded = "model_context_window_exceeded"
	FinishReasonNetworkError               = "network_error"
)

// ResponseMsg is the assistant message in a response choice.
type ResponseMsg struct {
	Role             string     `json:"role"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall is a tool invocation requested by the model. In streamed deltas,
// Index identifies which call a fragment belongs to: concatenate
// Function.Arguments fragments that share an Index.
type ToolCall struct {
	Index    int           `json:"index,omitempty"`
	ID       string        `json:"id,omitempty"`
	Type     string        `json:"type,omitempty"`
	Function *FunctionCall `json:"function,omitempty"`
	MCP      *MCPCall      `json:"mcp,omitempty"`
}

// FunctionCall is a function invocation with its JSON arguments string.
type FunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// MCPCall reports a server-side MCP interaction: a tool listing
// (Type "mcp_list_tools", Tools set) or a tool call (Type "mcp_call", with
// Name, Arguments, and Output or Error).
type MCPCall struct {
	ID          string `json:"id,omitempty"`
	Type        string `json:"type,omitempty"`
	ServerLabel string `json:"server_label,omitempty"`
	Name        string `json:"name,omitempty"`
	Arguments   string `json:"arguments,omitempty"`
	Output      any    `json:"output,omitempty"`
	Tools       []any  `json:"tools,omitempty"`
	Error       string `json:"error,omitempty"`
}

// Usage is token usage. In a stream it arrives on the final chunk only.
type Usage struct {
	PromptTokens            int                     `json:"prompt_tokens"`
	CompletionTokens        int                     `json:"completion_tokens"`
	TotalTokens             int                     `json:"total_tokens"`
	PromptTokensDetails     *PromptTokensDetail     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *CompletionTokensDetail `json:"completion_tokens_details,omitempty"`
}

// PromptTokensDetail breaks down prompt tokens.
type PromptTokensDetail struct {
	CachedTokens int `json:"cached_tokens"` // served from the context cache
}

// CompletionTokensDetail breaks down completion tokens.
type CompletionTokensDetail struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

// StreamDelta is the incremental content of one streamed chunk.
type StreamDelta struct {
	Role             string     `json:"role,omitempty"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

// StreamChoice is one choice within a streamed chunk.
type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

// StreamChunk is one server-sent-event payload of a streaming completion.
type StreamChunk struct {
	ID        string         `json:"id"`
	RequestID string         `json:"request_id,omitempty"`
	Created   int64          `json:"created,omitempty"`
	Model     string         `json:"model"`
	Choices   []StreamChoice `json:"choices"`
	Usage     *Usage         `json:"usage,omitempty"`
}
