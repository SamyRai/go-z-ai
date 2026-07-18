package client

import (
	"context"
	"fmt"
)

// EmbeddingsService generates vector embeddings from text, against
// BigModelBaseURL (open.bigmodel.cn) — the only platform that documents
// this endpoint (docs.z.ai's doc index has no Embeddings API at all).
// Request/response types confirmed against the live OpenAPI spec at
// https://docs.bigmodel.cn/openapi/openapi.json (POST /paas/v4/embeddings)
// and cross-checked against the official Java SDK's typed models
// (github.com/zai-org/z-ai-sdk-java, service/embedding/*.java), which match
// field-for-field (unlike Moderations — see ModerationsService). No public
// repo has recorded HTTP fixtures/VCR cassettes for this endpoint: searched
// the Python and Java SDKs' test suites (live-integration-only, gated on a
// real API key, no stubs) and GitHub code search broadly, nothing found.
// Both embedding-2 and embedding-3 currently return 400 "Unknown Model"
// (code 1211) on ProdBaseURL AND on BigModelBaseURL for at least one
// GLM-Coding-Plan account, live-verified 2026-07-11 to be an
// account/plan-entitlement gate rather than a platform-routing issue (that
// account's key authenticates fine on both platforms — same /models
// catalog, same chat/completions billing error — it just has no embedding
// models in its catalog). See BigModelBaseURL's doc comment and
// docs/en/accounts-and-quota.md.
type EmbeddingsService struct {
	client *Client
}

// Embedding model constants (docs.bigmodel.cn, 2026-07-11). embedding-2 has
// a fixed 1024-dim output; embedding-3 defaults to 2048 and accepts
// EmbeddingsRequest.Dimensions of 256/512/1024/2048.
const (
	EmbeddingModel2 = "embedding-2"
	EmbeddingModel3 = "embedding-3"
)

// EmbeddingsRequest requests one or more text embeddings. Input is a string
// or a []string (batched) — embedding-2 caps a single input at 512 tokens
// and an array at 8K tokens total; embedding-3 caps a single input at 3072
// tokens and an array at 64 items.
type EmbeddingsRequest struct {
	Model      string `json:"model"`
	Input      any    `json:"input"`
	Dimensions int    `json:"dimensions,omitempty"`
}

// EmbeddingObject is one input's embedding result, positioned by Index to
// match its place in a batched Input array.
type EmbeddingObject struct {
	Index     int       `json:"index"`
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
}

// EmbeddingsResponse is the result of EmbeddingsService.Create.
type EmbeddingsResponse struct {
	Object string            `json:"object"`
	Data   []EmbeddingObject `json:"data"`
	Model  string            `json:"model"`
	Usage  Usage             `json:"usage"`
}

// Create generates embeddings for req.Input (a string or []string) using
// req.Model (EmbeddingModel2 or EmbeddingModel3). Authenticates with
// Config.ChinaAPIKey (falling back to Config.APIKey) against
// BigModelBaseURL, independent of Config.BaseURL.
func (s *EmbeddingsService) Create(ctx context.Context, req EmbeddingsRequest) (*EmbeddingsResponse, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if req.Input == nil {
		return nil, fmt.Errorf("input is required")
	}

	var resp EmbeddingsResponse
	apiKey := s.client.chinaAPIKey()
	if err := s.client.doRequestBaseKey(ctx, BigModelBaseURL, apiKey, "POST", "/embeddings", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}
	return &resp, nil
}
