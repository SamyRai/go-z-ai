package client

import (
	"context"
	"fmt"
)

// ToolsService handles Z.AI tool capabilities
type ToolsService struct {
	client *Client
}

// WebSearchRequest represents a web search request
type WebSearchRequest struct {
	Query  string `json:"query"`
	TopK   int    `json:"top_k,omitempty"`
	Stream bool   `json:"stream,omitempty"`
}

// WebSearchResponse represents web search results
type WebSearchResponse struct {
	Code    int               `json:"code"`
	Msg     string            `json:"msg"`
	Data    []WebSearchResult `json:"data,omitempty"`
	Success bool              `json:"success"`
}

// WebSearchResult represents a single search result
type WebSearchResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score,omitempty"`
}

// WebReaderRequest represents a web reader request
type WebReaderRequest struct {
	URL         string `json:"url"`
	WithImages  bool   `json:"with_images,omitempty"`
	WithSummary bool   `json:"with_summary,omitempty"`
	WithLinks   bool   `json:"with_links,omitempty"`
}

// WebReaderResponse represents web reader results
type WebReaderResponse struct {
	Code    int            `json:"code"`
	Msg     string         `json:"msg"`
	Data    *WebReaderData `json:"data,omitempty"`
	Success bool           `json:"success"`
}

// WebReaderData represents parsed web page data
type WebReaderData struct {
	Title    string            `json:"title"`
	Content  string            `json:"content"`
	URL      string            `json:"url"`
	Images   []string          `json:"images,omitempty"`
	Links    []string          `json:"links,omitempty"`
	Summary  string            `json:"summary,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// TokenizerRequest represents a tokenization request
type TokenizerRequest struct {
	Text  string `json:"text"`
	Model string `json:"model,omitempty"`
}

// TokenizerResponse represents tokenization results
type TokenizerResponse struct {
	Code    int            `json:"code"`
	Msg     string         `json:"msg"`
	Data    *TokenizerData `json:"data,omitempty"`
	Success bool           `json:"success"`
}

// TokenizerData represents tokenization data
type TokenizerData struct {
	TokenCount int      `json:"token_count"`
	Tokens     []string `json:"tokens,omitempty"`
}

// NewToolsService creates a new tools service
func NewToolsService(client *Client) *ToolsService {
	return &ToolsService{client: client}
}

// WebSearch performs web search using Z.AI's specialized search engine
func (s *ToolsService) WebSearch(ctx context.Context, query string, topK int) (*WebSearchResponse, error) {
	req := WebSearchRequest{Query: query, TopK: topK}
	var result WebSearchResponse
	if err := s.client.doRequestBase(ctx, ToolsBaseURL, "POST", "/web/search", req, &result); err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	return &result, nil
}

// WebReader reads and parses content from a URL
func (s *ToolsService) WebReader(ctx context.Context, url string, withImages, withSummary bool) (*WebReaderResponse, error) {
	req := WebReaderRequest{URL: url, WithImages: withImages, WithSummary: withSummary, WithLinks: true}
	var result WebReaderResponse
	if err := s.client.doRequestBase(ctx, ToolsBaseURL, "POST", "/web/reader", req, &result); err != nil {
		return nil, fmt.Errorf("failed to read url: %w", err)
	}
	return &result, nil
}

// Tokenize counts tokens for a given text
func (s *ToolsService) Tokenize(ctx context.Context, text, model string) (*TokenizerResponse, error) {
	req := TokenizerRequest{Text: text, Model: model}
	var result TokenizerResponse
	if err := s.client.doRequestBase(ctx, ToolsBaseURL, "POST", "/tokenizer", req, &result); err != nil {
		return nil, fmt.Errorf("failed to tokenize: %w", err)
	}
	return &result, nil
}
