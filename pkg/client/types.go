package client

import (
	"encoding/json"
	"time"
)

// Message represents a chat message. Content is plain text for ordinary use;
// set Images (one or more https:// URLs or data: URIs) to attach images for
// vision models (GLM-4.6V/4.5V) — MarshalJSON/UnmarshalJSON handle the
// content-parts wire shape transparently, so existing Content-only code is
// unaffected. See content.go.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Images     []string   `json:"-"`                      // image URLs/data-URIs; non-empty switches Content to a content-parts array on the wire
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // set on assistant messages that request tool calls
	ToolCallID string     `json:"tool_call_id,omitempty"` // set on role:"tool" messages to reference the call
	Name       string     `json:"name,omitempty"`         // optional tool/function name
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	Temperature    float64         `json:"temperature,omitempty"`
	TopP           float64         `json:"top_p,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	Stream         bool            `json:"stream,omitempty"`
	DoSample       bool            `json:"do_sample,omitempty"`
	Stop           []string        `json:"stop,omitempty"`
	Tools          []Tool          `json:"tools,omitempty"`
	ToolChoice     string          `json:"tool_choice,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
	Thinking       *ThinkingConfig `json:"thinking,omitempty"`
	Metadata       map[string]any  `json:"metadata,omitempty"`
	User           string          `json:"user,omitempty"`
	// StreamToolCall enables streaming responses for function (tool) calls,
	// so tool-call deltas arrive incrementally like content deltas instead of
	// in a single batch at the end. Default false. Only supported by GLM-4.6
	// and above. NOT VERIFIED LIVE — captured from the docs.z.ai
	// chat-completion spec; record a cassette to confirm the SSE chunk shape
	// before relying on it. See StreamDelta.ToolCalls for where the deltas land.
	StreamToolCall bool `json:"stream_tool_call,omitempty"`
}

// Effort levels for ThinkingConfig.Effort, as documented on docs.z.ai. The
// server normalizes: none/minimal -> skip thinking, low/medium -> high,
// xhigh -> max. Effort is only honored by GLM-5.2; other models ignore it.
const (
	EffortMax     = "max"
	EffortXhigh   = "xhigh"
	EffortHigh    = "high"
	EffortMedium  = "medium"
	EffortLow     = "low"
	EffortMinimal = "minimal"
	EffortNone    = "none"
)

// ThinkingConfig controls chain-of-thought reasoning
type ThinkingConfig struct {
	Type      string `json:"type,omitempty"`      // enabled, disabled
	Preserved bool   `json:"preserved,omitempty"` // keep reasoning across turns
	Effort    string `json:"effort,omitempty"`    // one of the Effort* constants (GLM-5.2 only)
}

// ResponseFormat controls the output format
type ResponseFormat struct {
	Type string `json:"type"` // text, json_object, json_schema
	// JSONSchema is required when Type == "json_schema" (structured output).
	JSONSchema *JSONSchemaConfig `json:"json_schema,omitempty"`
}

// JSONSchemaConfig configures structured (json_schema) output. Schema is passed
// through verbatim as a raw JSON blob so callers can supply any valid schema
// without it being modeled here.
type JSONSchemaConfig struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
	Strict bool            `json:"strict,omitempty"`
}

// NewJSONSchemaFormat builds a ResponseFormat for structured (json_schema) output
// from a raw JSON schema blob, the preferred form for CLI/file-driven schemas.
func NewJSONSchemaFormat(name string, schema json.RawMessage, strict bool) *ResponseFormat {
	return &ResponseFormat{
		Type:       "json_schema",
		JSONSchema: &JSONSchemaConfig{Name: name, Schema: schema, Strict: strict},
	}
}

// Tool type identifiers, as documented on docs.z.ai. The chat-completion
// spec lists "function", "retrieval", and "web_search" as the possible
// tools[].type values — though the same spec also notes only "function" is
// fully supported today. The retrieval/web_search request shapes are marked
// NOT VERIFIED LIVE until a cassette pins them down; see NewRetrievalTool /
// NewWebSearchTool.
const (
	ToolTypeFunction  = "function"
	ToolTypeRetrieval = "retrieval"
	ToolTypeWebSearch = "web_search"
)

// ToolMaxFunctions is the documented cap on the number of tools a single
// request may carry when any are type "function". Enforced in
// validateChatRequest so callers get a clear client-side error rather than
// an opaque server one.
const ToolMaxFunctions = 128

// Tool represents a tool the model may invoke. Type selects which payload
// field the server reads; today only Type=="function" (Function) is
// live-supported. The wire shape of retrieval/web_search is modeled for
// forward compatibility and marked NOT VERIFIED LIVE.
type Tool struct {
	Type      string       `json:"type"`
	Function  *FunctionDef `json:"function,omitempty"`
	Retrieval *Retrieval   `json:"retrieval,omitempty"`
	WebSearch *WebSearch   `json:"web_search,omitempty"`
}

// FunctionDef defines a function for tool calling. Name must match
// ^[a-zA-Z0-9_-]+$ and be 1–64 chars (enforced in validateChatRequest).
type FunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// Retrieval is the payload for a type:"retrieval" tool. NOT VERIFIED LIVE —
// the chat-completion spec documents this tool type but its request shape is
// not yet pinned by a cassette.
type Retrieval struct {
	Prompt      string `json:"prompt,omitempty"`
	KnowledgeID string `json:"knowledge_id,omitempty"`
}

// WebSearch is the payload for a type:"web_search" tool. The shape follows
// the official Python SDK example (tools=[{"type":"web_search","web_search":
// {"search_query":["..."]}}]). NOT VERIFIED LIVE against a chat-completion
// success response — SearchQuery is the documented field; additional fields
// the server may accept are not modeled here to avoid guessing wire names.
type WebSearch struct {
	SearchQuery []string `json:"search_query,omitempty"`
}

// NewFunctionTool is the typed constructor for a function tool.
func NewFunctionTool(name, description string, parameters map[string]any) Tool {
	return Tool{Type: ToolTypeFunction, Function: &FunctionDef{
		Name: name, Description: description, Parameters: parameters,
	}}
}

// NewRetrievalTool is the typed constructor for a retrieval tool. NOT VERIFIED
// LIVE.
func NewRetrievalTool(knowledgeID, prompt string) Tool {
	return Tool{Type: ToolTypeRetrieval, Retrieval: &Retrieval{KnowledgeID: knowledgeID, Prompt: prompt}}
}

// NewWebSearchTool is the typed constructor for a web_search tool carrying one
// or more search queries. NOT VERIFIED LIVE.
func NewWebSearchTool(queries ...string) Tool {
	return Tool{Type: ToolTypeWebSearch, WebSearch: &WebSearch{SearchQuery: queries}}
}

// ChatResponse represents a chat completion response. When the request used a
// web_search tool, WebSearch carries the matched sources the model grounded
// its answer in (a top-level array, sibling to Choices — the entry shape
// reuses WebSearchResult from tools.go, which matches docs.z.ai's web-search
// reference; NOT VERIFIED LIVE until a chat-completion cassette pins the exact
// placement of the array).
type ChatResponse struct {
	ID        string            `json:"id"`
	Created   int64             `json:"created"`
	Model     string            `json:"model"`
	Choices   []Choice          `json:"choices"`
	Usage     Usage             `json:"usage"`
	WebSearch []WebSearchResult `json:"web_search,omitempty"`
}

// Choice represents a response choice
type Choice struct {
	Index        int         `json:"index"`
	Message      ResponseMsg `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// FinishReason values documented on docs.z.ai. Code that needs to switch on
// the reason should prefer these consts over bare strings.
const (
	FinishReasonStop                       = "stop"
	FinishReasonToolCalls                  = "tool_calls"
	FinishReasonLength                     = "length"
	FinishReasonSensitive                  = "sensitive"
	FinishReasonModelContextWindowExceeded = "model_context_window_exceeded"
	FinishReasonNetworkError               = "network_error"
)

// ResponseMsg represents the response message
type ResponseMsg struct {
	Role             string     `json:"role"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool/function call
type ToolCall struct {
	ID       string        `json:"id,omitempty"`
	Type     string        `json:"type,omitempty"`
	Function *FunctionCall `json:"function,omitempty"`
}

// FunctionCall represents a function call with arguments
type FunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens        int                 `json:"prompt_tokens"`
	CompletionTokens    int                 `json:"completion_tokens"`
	TotalTokens         int                 `json:"total_tokens"`
	PromptTokensDetails *PromptTokensDetail `json:"prompt_tokens_details,omitempty"`
}

// PromptTokensDetail contains detailed prompt token information
type PromptTokensDetail struct {
	CachedTokens int `json:"cached_tokens"`
}

// ModelsInfo represents available models information
type ModelsInfo struct {
	Models []ModelDetails `json:"data"`
}

// ModelDetails represents individual model information
type ModelDetails struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ContextSize int      `json:"max_context"`
	OwnedBy     string   `json:"owned_by"`
	Pricing     *Pricing `json:"pricing,omitempty"`
}

// Pricing represents model pricing information
type Pricing struct {
	Input      float64 `json:"prompt"`
	Output     float64 `json:"completion"`
	Cached     float64 `json:"cached_prompt,omitempty"`
	CacheStore float64 `json:"cached_prompt_storage,omitempty"`
	Unit       string  `json:"unit,omitempty"`
}

// UsageInfo represents current usage and quota information
type UsageInfo struct {
	QuotaID        string    `json:"quota_id,omitempty"`
	TotalQuota     int64     `json:"total_quota"`
	UsedQuota      int64     `json:"used_quota"`
	RemainingQuota int64     `json:"remaining_quota"`
	ResetTime      time.Time `json:"reset_time"`
	UsagePeriod    string    `json:"usage_period"` // 5hour, weekly
}

// AccountInfo represents account information
type AccountInfo struct {
	AccountID   string       `json:"account_id"`
	AccountType string       `json:"account_type"` // payg, subscription, enterprise
	PlanName    string       `json:"plan_name,omitempty"`
	Status      string       `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	UsageStats  *UsageStats  `json:"usage_stats,omitempty"`
	BillingInfo *BillingInfo `json:"billing_info,omitempty"`
}

// UsageStats represents account usage statistics
type UsageStats struct {
	TotalRequests      int64     `json:"total_requests"`
	TotalTokens        int64     `json:"total_tokens"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
}

// BillingInfo represents billing information
type BillingInfo struct {
	BillingCycle   string    `json:"billing_cycle"` // monthly, yearly
	NextBillDate   time.Time `json:"next_bill_date"`
	LastBillAmount float64   `json:"last_bill_amount,omitempty"`
	Currency       string    `json:"currency"`
}

// --- Streaming types ---

// StreamDelta is the incremental content delivered in a single streamed chunk.
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

// StreamChunk is one Server-Sent-Event payload from a streaming completion.
type StreamChunk struct {
	ID      string         `json:"id"`
	Created int64          `json:"created,omitempty"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
	Usage   *Usage         `json:"usage,omitempty"`
}
