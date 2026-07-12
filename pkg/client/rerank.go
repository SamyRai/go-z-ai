package client

import (
	"context"
	"fmt"
)

// RerankService scores a set of candidate documents against a query for
// relevance — used in RAG/retrieval pipelines to reorder search results
// before feeding them to a chat model. Confirmed against docs.bigmodel.cn's
// live OpenAPI spec (https://docs.bigmodel.cn/openapi/openapi.json,
// POST /paas/v4/rerank). Distinct from Embeddings: rerank directly scores
// query/document pairs rather than producing vectors to compare separately.
type RerankService struct {
	client *Client
}

// RerankModel is the only value RerankRequest.Model currently accepts.
const RerankModel = "rerank"

// RerankRequest scores req.Documents against req.Query. Model, Query, and
// Documents are required.
type RerankRequest struct {
	Model           string   `json:"model"`           // defaults to RerankModel when empty
	Query           string   `json:"query"`           // max 4096 chars
	Documents       []string `json:"documents"`       // max 128 items, each max 4096 chars
	TopN            int      `json:"top_n,omitempty"` // top N results by score; 0/omitted returns all
	ReturnDocuments bool     `json:"return_documents,omitempty"`
	ReturnRawScores bool     `json:"return_raw_scores,omitempty"`
	RequestID       string   `json:"request_id,omitempty"`
	UserID          string   `json:"user_id,omitempty"`
}

// RerankResult is one scored document in RerankResponse.Results.
type RerankResult struct {
	Document       string  `json:"document"` // populated only when ReturnDocuments was true
	Index          int     `json:"index"`    // position in the original Documents array
	RelevanceScore float64 `json:"relevance_score"`
}

// RerankUsage is token usage for a rerank call.
type RerankUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// RerankResponse is the result of RerankService.Create.
type RerankResponse struct {
	ID        string         `json:"id"`
	Created   int64          `json:"created"`
	RequestID string         `json:"request_id"`
	Results   []RerankResult `json:"results"`
	Usage     RerankUsage    `json:"usage"`
}

// Create scores req.Documents against req.Query, returning results ordered
// by relevance. req.Model defaults to RerankModel ("rerank", currently the
// only valid value) when empty.
func (s *RerankService) Create(ctx context.Context, req RerankRequest) (*RerankResponse, error) {
	if req.Model == "" {
		req.Model = RerankModel
	}
	if req.Query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if len(req.Documents) == 0 {
		return nil, fmt.Errorf("at least one document is required")
	}

	var resp RerankResponse
	if err := s.client.doRequest(ctx, "POST", "/rerank", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to rerank documents: %w", err)
	}
	return &resp, nil
}
