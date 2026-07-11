package client

import (
	"context"
	"fmt"
)

// AgentsBaseURL is the base URL for the Agents API. Verified live: a bare
// root path — nesting /v1/agents under the chat-completions BaseURL (either
// ProdBaseURL or CodingBaseURL) 404s (confirmed via a real call:
// api.z.ai/api/coding/paas/v4/v1/agents -> 404, api.z.ai/api/v1/agents -> 200).
const AgentsBaseURL = "https://api.z.ai/api"

// AgentsService invokes Z.AI's specialized agents (e.g. "general_translation",
// GLM Slide/Poster, Video Effect Template), each identified by an agent_id,
// over the shared /v1/agents endpoint.
type AgentsService struct {
	client *Client
}

// AgentMessage is one message in an agent invocation. Content is always a
// parts array on this endpoint (verified live) — unlike ChatService's
// Message, there is no plain-string shorthand for content.
type AgentMessage struct {
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
}

// NewAgentTextMessage builds a single-part text AgentMessage — the common
// case for a plain-text prompt.
func NewAgentTextMessage(role, text string) AgentMessage {
	return AgentMessage{Role: role, Content: []ContentPart{{Type: "text", Text: text}}}
}

// AgentInvokeRequest invokes a named agent.
type AgentInvokeRequest struct {
	AgentID         string         `json:"agent_id"`
	Messages        []AgentMessage `json:"messages"`
	RequestID       string         `json:"request_id,omitempty"`
	UserID          string         `json:"user_id,omitempty"`
	Stream          bool           `json:"stream,omitempty"`
	CustomVariables map[string]any `json:"custom_variables,omitempty"`
}

// AgentError describes why an agent invocation failed. Verified live: this
// arrives inside an HTTP 200 response body, not via a non-200 HTTP status —
// see AgentResponse.Failed.
type AgentError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// AgentContent is one content part of an agent's reply message.
type AgentContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// AgentReplyMessage is the assistant's reply within an AgentChoice.
// NOT VERIFIED LIVE: modeled from docs.z.ai's documented example only — the
// account used for live verification had insufficient balance to reach a
// successful invocation, so only the failure envelope (ID/AgentID/Status/
// Error below) was confirmed against a real recorded response. See
// testdata/cassettes/agents_invoke.yaml.
type AgentReplyMessage struct {
	Role    string       `json:"role"`
	Content AgentContent `json:"content"`
}

// AgentChoice is one response choice from a non-streaming agent invocation.
// NOT VERIFIED LIVE — see AgentReplyMessage.
type AgentChoice struct {
	Index        int               `json:"index"`
	FinishReason string            `json:"finish_reason,omitempty"`
	Messages     AgentReplyMessage `json:"messages"`
}

// AgentUsage is token usage for an agent invocation. NOT VERIFIED LIVE.
type AgentUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	TotalCalls       int `json:"total_calls,omitempty"`
}

// AgentResponse is the result of invoking an agent. ID/AgentID/Status/Error
// are verified against a real recorded response (testdata/cassettes/
// agents_invoke.yaml); Choices/Usage are modeled from docs.z.ai's example
// only and not yet confirmed live.
type AgentResponse struct {
	ID      string        `json:"id"`
	AgentID string        `json:"agent_id"`
	Status  string        `json:"status,omitempty"` // e.g. "failed"; absent on the (unverified) success shape
	Choices []AgentChoice `json:"choices,omitempty"`
	Usage   *AgentUsage   `json:"usage,omitempty"`
	Error   *AgentError   `json:"error,omitempty"`
}

// Failed reports whether the invocation failed at the business level. The
// Agents API returns HTTP 200 even on failure (verified live — e.g.
// insufficient balance) — callers MUST check this (or Error/Status
// directly) rather than relying on Invoke returning a non-nil error, which
// only signals a transport/HTTP-level failure.
func (r *AgentResponse) Failed() bool {
	return r.Status == "failed" || r.Error != nil
}

// Invoke calls a named agent (e.g. "general_translation"). A non-nil error
// return means the HTTP request itself failed; check resp.Failed() for a
// business-level failure the API reports inside a 200 OK body.
func (s *AgentsService) Invoke(ctx context.Context, req AgentInvokeRequest) (*AgentResponse, error) {
	if req.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("at least one message is required")
	}

	var resp AgentResponse
	if err := s.client.doRequestBase(ctx, AgentsBaseURL, "POST", "/v1/agents", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to invoke agent: %w", err)
	}
	return &resp, nil
}
