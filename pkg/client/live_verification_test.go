package client

import (
	"context"
	"testing"
)

// These two tests replay real recorded interactions from a live account
// (2026-07-10, coding_plan type) against the general PAYG base
// (ProdBaseURL) to lock in a confirmed finding: both endpoints are reachable
// and correctly routed, but reject the model names documented in the
// official SDK's own test fixtures ("moderation", "embedding-2") with
// error 1211 "Unknown Model". This rules out a base-URL/routing problem —
// the remaining unknown is the correct live model code (or an account/plan
// gate), which needs further live probing before a Moderations/Embeddings
// service can be implemented with confidence. See todo.md.
//
// No service type exists yet for either endpoint (that's the point — we
// don't have a confirmed success response to model), so these replay
// directly through doRequestBase and assert on the parsed *APIError.

func TestModerationsLiveErrorIsUnknownModelNotRouting(t *testing.T) {
	c := newReplayClient(t, "moderations", ProdBaseURL)

	var result any
	err := c.doRequestBase(context.Background(), ProdBaseURL, "POST", "/moderations", map[string]any{
		"model": "moderation",
		"input": "hello world",
	}, &result)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError (confirms the endpoint is real and routes correctly), got %T: %v", err, err)
	}
	if apiErr.Code != 1211 {
		t.Errorf("expected code 1211 (Unknown Model), got %d: %s", apiErr.Code, apiErr.Message)
	}
}

func TestEmbeddingsLiveErrorIsUnknownModelNotRouting(t *testing.T) {
	c := newReplayClient(t, "embeddings", ProdBaseURL)

	var result any
	err := c.doRequestBase(context.Background(), ProdBaseURL, "POST", "/embeddings", map[string]any{
		"model": "embedding-2",
		"input": "hello",
	}, &result)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError (confirms the endpoint is real and routes correctly), got %T: %v", err, err)
	}
	if apiErr.Code != 1211 {
		t.Errorf("expected code 1211 (Unknown Model), got %d: %s", apiErr.Code, apiErr.Message)
	}
}
