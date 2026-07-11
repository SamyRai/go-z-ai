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
	Model          string                 `json:"model"`
	Messages       []Message              `json:"messages"`
	Temperature    float64                `json:"temperature,omitempty"`
	TopP           float64                `json:"top_p,omitempty"`
	MaxTokens      int                    `json:"max_tokens,omitempty"`
	Stream         bool                   `json:"stream,omitempty"`
	DoSample       bool                   `json:"do_sample,omitempty"`
	Stop           []string               `json:"stop,omitempty"`
	Tools          []Tool                 `json:"tools,omitempty"`
	ToolChoice     string                 `json:"tool_choice,omitempty"`
	ResponseFormat *ResponseFormat        `json:"response_format,omitempty"`
	Thinking       *ThinkingConfig        `json:"thinking,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	User           string                 `json:"user,omitempty"`
}

// ThinkingConfig controls chain-of-thought reasoning
type ThinkingConfig struct {
	Type      string `json:"type,omitempty"`      // enabled, disabled
	Preserved bool   `json:"preserved,omitempty"` // keep reasoning across turns
	Effort    string `json:"effort,omitempty"`    // max, high, medium, low, minimal, none
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

// Tool represents a function/tool definition
type Tool struct {
	Type     string       `json:"type"` // function
	Function *FunctionDef `json:"function,omitempty"`
}

// FunctionDef defines a function for tool calling
type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID      string   `json:"id"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a response choice
type Choice struct {
	Index        int         `json:"index"`
	Message      ResponseMsg `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

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
