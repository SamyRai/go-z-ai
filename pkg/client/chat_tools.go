package client

import (
	"context"
	"fmt"
)

// ToolMaxRounds bounds RunWithTools against runaway tool-calling loops.
const ToolMaxRounds = 8

// ToolExecutor executes a single tool call: it receives the function name and
// the raw JSON arguments string the model produced, and returns the tool's
// result (placed verbatim into the next role:"tool" message). A returned error
// is reported to the model as the tool's outcome so it can recover gracefully.
type ToolExecutor func(name, arguments string) (string, error)

// RunWithTools performs a chat completion that transparently executes any
// function/tool calls the model makes. When the model responds with
// finish_reason="tool_calls", each call is dispatched to exec; the assistant
// message and tool results are appended, and the request is repeated — until
// the model returns a non-tool finish or ToolMaxRounds is exceeded.
func (s *ChatService) RunWithTools(ctx context.Context, req ChatRequest, exec ToolExecutor) (*ChatResponse, error) {
	return s.RunWithToolsLimit(ctx, req, exec, ToolMaxRounds)
}

// RunWithToolsLimit is RunWithTools with an explicit round cap.
func (s *ChatService) RunWithToolsLimit(ctx context.Context, req ChatRequest, exec ToolExecutor, maxRounds int) (*ChatResponse, error) {
	if err := validateChatRequest(&req); err != nil {
		return nil, fmt.Errorf("invalid chat request: %w", err)
	}
	if len(req.Tools) == 0 {
		return nil, fmt.Errorf("RunWithTools requires at least one tool in req.Tools")
	}
	if exec == nil {
		return nil, fmt.Errorf("RunWithTools requires a non-nil executor")
	}
	if maxRounds <= 0 {
		maxRounds = ToolMaxRounds
	}

	// Work on a copy of the messages so the caller's slice is not mutated.
	messages := make([]Message, len(req.Messages))
	copy(messages, req.Messages)

	var resp *ChatResponse
	for round := 0; round < maxRounds; round++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		req.Messages = messages
		r, err := s.Create(req)
		if err != nil {
			return nil, err
		}
		resp = r
		if len(r.Choices) == 0 {
			return r, nil
		}
		choice := r.Choices[0]
		if choice.FinishReason != "tool_calls" || len(choice.Message.ToolCalls) == 0 {
			return r, nil
		}

		// Echo the assistant's tool-call message, then append each tool result.
		messages = append(messages, Message{
			Role:      "assistant",
			Content:   choice.Message.Content,
			ToolCalls: choice.Message.ToolCalls,
		})
		for _, call := range choice.Message.ToolCalls {
			name, args := "", ""
			if call.Function != nil {
				name = call.Function.Name
				args = call.Function.Arguments
			}
			result, execErr := exec(name, args)
			if execErr != nil {
				result = fmt.Sprintf("error: %v", execErr)
			}
			messages = append(messages, Message{
				Role:       "tool",
				ToolCallID: call.ID,
				Name:       name,
				Content:    result,
			})
		}
	}
	return resp, fmt.Errorf("tool-calling loop exceeded %d rounds without a final answer", maxRounds)
}
