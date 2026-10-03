package client

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
)

// ResponsesService calls Z.AI's OpenAI Responses-protocol endpoint, the
// surface Codex speaks (see responses_types.go). Region selects the gateway:
// https://api.z.ai/api/v1 or https://open.bigmodel.cn/api/v1.
type ResponsesService struct {
	client *Client
}

const responsesEndpoint = "/responses"

// Create sends a non-streaming request. A response whose status is failed
// is returned as an *APIError.
func (s *ResponsesService) Create(ctx context.Context, req ResponsesRequest) (*ResponsesResponse, error) {
	req.Stream = false
	r, err := s.prepare(req)
	if err != nil {
		return nil, err
	}
	var resp ResponsesResponse
	if err := s.client.do(ctx, r, &resp); err != nil {
		return nil, fmt.Errorf("failed to create response: %w", err)
	}
	if err := responsesFailure(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Stream sends a streaming request and returns an iterator over its events.
// Iterator semantics match ChatService.Stream; an error event or a failed
// response ends the stream with an *APIError.
func (s *ResponsesService) Stream(ctx context.Context, req ResponsesRequest) iter.Seq2[ResponsesStreamEvent, error] {
	req.Stream = true
	r, err := s.prepare(req)
	if err != nil {
		return errorIter[ResponsesStreamEvent](err)
	}
	return streamSSE(ctx, s.client, r, decodeResponsesStream)
}

// prepare validates req, applies the tool-schema compatibility rewrite, and
// wraps it as an apiRequest against the region's Responses root.
func (s *ResponsesService) prepare(req ResponsesRequest) (apiRequest, error) {
	if err := validateResponsesRequest(&req); err != nil {
		return apiRequest{}, fmt.Errorf("invalid responses request: %w", err)
	}
	if !s.client.config.DisableToolSchemaCompat {
		req.Tools = sanitizeTools(req.Tools, func(t *ResponsesTool) *map[string]any { return &t.Parameters })
	}
	return apiRequest{
		method:  "POST",
		baseURL: s.client.config.Region.ResponsesBaseURL(),
		path:    responsesEndpoint,
		body:    req,
		service: "responses",
		model:   req.Model,
	}, nil
}

// validateResponsesRequest checks req client-side and defaults tool types.
func validateResponsesRequest(req *ResponsesRequest) error {
	if req.Model == "" {
		return errors.New("model is required")
	}
	if len(req.Input) == 0 {
		return errors.New("input is required")
	}
	if req.Reasoning != nil {
		if err := validateEffort(req.Model, req.Reasoning.Effort); err != nil {
			return err
		}
	}
	for i := range req.Tools {
		t := &req.Tools[i]
		t.Type = cmp.Or(t.Type, ToolTypeFunction)
		if t.Type == ToolTypeFunction && !toolNamePattern.MatchString(t.Name) {
			return fmt.Errorf("tools[%d].name %q must match %s", i, t.Name, toolNamePattern)
		}
	}
	return nil
}

// decodeResponsesStream decodes the Responses SSE stream into events. The
// event type comes from the payload's "type" (falling back to the SSE event
// name); error and response.failed end the stream with an *APIError.
func decodeResponsesStream(ctx context.Context, body io.Reader, emit func(ResponsesStreamEvent) error) error {
	err := scanSSE(ctx, body, func(event, data string) error {
		switch data {
		case "":
			return nil
		case sseDone:
			return errStreamDone
		}
		var ev ResponsesStreamEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			return fmt.Errorf("failed to parse responses event: %w", err)
		}
		ev.Type = cmp.Or(ev.Type, event)
		switch ev.Type {
		case "error":
			return errorFromBody(http.StatusOK, []byte(data))
		case "response.failed":
			if err := responsesFailure(ev.Response); err != nil {
				return err
			}
			return createAPIError(0, http.StatusOK, "response failed")
		}
		return emit(ev)
	})
	if errors.Is(err, errStreamDone) {
		return nil
	}
	return err
}

// responsesFailure returns the *APIError for a failed response, or nil.
func responsesFailure(r *ResponsesResponse) error {
	if r == nil || (r.Status != ResponsesStatusFailed && r.Error == nil) {
		return nil
	}
	msg := "response failed"
	if r.Error != nil {
		msg = cmp.Or(r.Error.Message, r.Error.Code, msg)
	}
	return createAPIError(0, http.StatusOK, msg)
}
