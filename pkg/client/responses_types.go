package client

import "strings"

// Wire types for POST /api/v1/responses, Z.AI's OpenAI Responses-protocol
// endpoint (the one Codex is configured against, docs.z.ai/devpack/tool/codex).
// They follow the OpenAI Responses API. NOT VERIFIED LIVE against Z.AI: the
// docs publish the endpoint but not its schema; pin it with a cassette (see
// TestVerifyResponses).

// Responses item types (ResponsesItem.Type).
const (
	ResponsesItemMessage            = "message"
	ResponsesItemFunctionCall       = "function_call"
	ResponsesItemFunctionCallOutput = "function_call_output"
	ResponsesItemReasoning          = "reasoning"
)

// Responses content-part types (ResponsesContent.Type).
const (
	ResponsesInputText     = "input_text"
	ResponsesInputImage    = "input_image"
	ResponsesOutputText    = "output_text"
	ResponsesSummaryText   = "summary_text"
	ResponsesReasoningText = "reasoning_text"
)

// Responses statuses (ResponsesResponse.Status).
const (
	ResponsesStatusCompleted  = "completed"
	ResponsesStatusIncomplete = "incomplete"
	ResponsesStatusFailed     = "failed"
)

// ResponsesRequest is a POST /responses body. Input is the conversation as
// items; Instructions plays the role of a system message.
type ResponsesRequest struct {
	Model             string              `json:"model"`
	Input             []ResponsesItem     `json:"input"`
	Instructions      string              `json:"instructions,omitempty"`
	Reasoning         *ResponsesReasoning `json:"reasoning,omitempty"`
	Tools             []ResponsesTool     `json:"tools,omitempty"`
	ToolChoice        string              `json:"tool_choice,omitempty"` // auto | none | required
	ParallelToolCalls *bool               `json:"parallel_tool_calls,omitempty"`
	MaxOutputTokens   int                 `json:"max_output_tokens,omitempty"`
	Temperature       *float64            `json:"temperature,omitempty"`
	TopP              *float64            `json:"top_p,omitempty"`
	Store             *bool               `json:"store,omitempty"`
	Stream            bool                `json:"stream,omitempty"`
}

// ResponsesReasoning configures reasoning. Effort takes the same levels as
// ChatRequest.ReasoningEffort and is validated against the catalog.
type ResponsesReasoning struct {
	Effort  string `json:"effort,omitempty"`
	Summary string `json:"summary,omitempty"` // auto | concise | detailed
}

// ResponsesTool is a function tool in the Responses API's flat shape.
// Parameters is a JSON Schema object; it gets the same GLM compatibility
// rewrite as chat tools unless Config.DisableToolSchemaCompat is set.
type ResponsesTool struct {
	Type        string         `json:"type"` // "function"; empty means function
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	Strict      *bool          `json:"strict,omitempty"`
}

// ResponsesItem is one conversation item, in a request's Input or a
// response's Output. Which fields apply depends on Type.
type ResponsesItem struct {
	Type   string `json:"type"`
	ID     string `json:"id,omitempty"`
	Status string `json:"status,omitempty"`
	// message
	Role    string             `json:"role,omitempty"` // user | system | developer | assistant
	Content []ResponsesContent `json:"content,omitempty"`
	// function_call and function_call_output
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"` // JSON text
	Output    string `json:"output,omitempty"`
	// reasoning
	Summary []ResponsesContent `json:"summary,omitempty"`
}

// ResponsesContent is one content part of an item.
type ResponsesContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"` // input_image: an https:// URL or data: URI
}

// ResponsesMessage builds a text message item. Assistant turns carry
// output_text parts; every other role carries input_text.
func ResponsesMessage(role, text string) ResponsesItem {
	partType := ResponsesInputText
	if role == "assistant" {
		partType = ResponsesOutputText
	}
	return ResponsesItem{Type: ResponsesItemMessage, Role: role, Content: []ResponsesContent{{Type: partType, Text: text}}}
}

// ResponsesFunctionOutput builds the item that returns a function call's
// result to the model.
func ResponsesFunctionOutput(callID, output string) ResponsesItem {
	return ResponsesItem{Type: ResponsesItemFunctionCallOutput, CallID: callID, Output: output}
}

// ResponsesResponse is a response object (POST /responses, or the payload of
// a response.completed stream event).
type ResponsesResponse struct {
	ID                string          `json:"id"`
	Object            string          `json:"object,omitempty"`
	CreatedAt         int64           `json:"created_at,omitempty"`
	Status            string          `json:"status,omitempty"`
	Model             string          `json:"model,omitempty"`
	Output            []ResponsesItem `json:"output,omitempty"`
	Usage             *ResponsesUsage `json:"usage,omitempty"`
	Error             *ResponsesError `json:"error,omitempty"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details,omitempty"`
}

// ResponsesUsage is a response's token accounting.
type ResponsesUsage struct {
	InputTokens         int                     `json:"input_tokens"`
	OutputTokens        int                     `json:"output_tokens"`
	TotalTokens         int                     `json:"total_tokens"`
	InputTokensDetails  *PromptTokensDetail     `json:"input_tokens_details,omitempty"`
	OutputTokensDetails *CompletionTokensDetail `json:"output_tokens_details,omitempty"`
}

// ResponsesError is a failed response's error.
type ResponsesError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// OutputText concatenates the text of the response's message items.
func (r *ResponsesResponse) OutputText() string {
	return r.joinParts(ResponsesItemMessage, func(it ResponsesItem) []ResponsesContent { return it.Content })
}

// ReasoningText concatenates the response's reasoning: summaries when the
// server sent them, else full reasoning text.
func (r *ResponsesResponse) ReasoningText() string {
	if s := r.joinParts(ResponsesItemReasoning, func(it ResponsesItem) []ResponsesContent { return it.Summary }); s != "" {
		return s
	}
	return r.joinParts(ResponsesItemReasoning, func(it ResponsesItem) []ResponsesContent { return it.Content })
}

// FunctionCalls returns the response's function_call items.
func (r *ResponsesResponse) FunctionCalls() []ResponsesItem {
	var calls []ResponsesItem
	for _, it := range r.Output {
		if it.Type == ResponsesItemFunctionCall {
			calls = append(calls, it)
		}
	}
	return calls
}

// joinParts concatenates the text of parts(item) across items of itemType.
func (r *ResponsesResponse) joinParts(itemType string, parts func(ResponsesItem) []ResponsesContent) string {
	var b strings.Builder
	for _, it := range r.Output {
		if it.Type != itemType {
			continue
		}
		for _, p := range parts(it) {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

// GetUsage implements usageBearer for the observability hooks, mapping the
// Responses token names onto Usage.
func (r *ResponsesResponse) GetUsage() *Usage {
	if r.Usage == nil {
		return nil
	}
	return &Usage{
		PromptTokens:            r.Usage.InputTokens,
		CompletionTokens:        r.Usage.OutputTokens,
		TotalTokens:             r.Usage.TotalTokens,
		PromptTokensDetails:     r.Usage.InputTokensDetails,
		CompletionTokensDetails: r.Usage.OutputTokensDetails,
	}
}

// Responses stream event types (ResponsesStreamEvent.Type) a client usually
// acts on; the server also sends lifecycle events such as response.created
// and response.output_item.added.
const (
	ResponsesEventOutputTextDelta       = "response.output_text.delta"
	ResponsesEventReasoningSummaryDelta = "response.reasoning_summary_text.delta"
	ResponsesEventReasoningTextDelta    = "response.reasoning_text.delta"
	ResponsesEventFunctionArgsDelta     = "response.function_call_arguments.delta"
	ResponsesEventOutputItemDone        = "response.output_item.done"
	ResponsesEventCompleted             = "response.completed"
	ResponsesEventIncomplete            = "response.incomplete"
)

// ResponsesStreamEvent is one streamed event. Delta carries text for the
// *.delta events, Item the finished item for response.output_item.done, and
// Response the final response for response.completed / response.incomplete.
// Failure events (error, response.failed) end the stream with an *APIError
// instead of being yielded.
type ResponsesStreamEvent struct {
	Type           string             `json:"type"`
	SequenceNumber int                `json:"sequence_number,omitempty"`
	OutputIndex    int                `json:"output_index,omitempty"`
	ItemID         string             `json:"item_id,omitempty"`
	Delta          string             `json:"delta,omitempty"`
	Item           *ResponsesItem     `json:"item,omitempty"`
	Response       *ResponsesResponse `json:"response,omitempty"`
}
