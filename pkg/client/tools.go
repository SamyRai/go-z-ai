package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ToolsService handles Z.AI tool capabilities
type ToolsService struct {
	client *Client
}

// WebSearchRequest represents a web search request
type WebSearchRequest struct {
	Query     string `json:"query"`
	TopK      int    `json:"top_k,omitempty"`
	Stream    bool   `json:"stream,omitempty"`
}

// WebSearchResponse represents web search results
type WebSearchResponse struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Data    []WebSearchResult `json:"data,omitempty"`
	Success bool   `json:"success"`
}

// WebSearchResult represents a single search result
type WebSearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
	Score   float64 `json:"score,omitempty"`
}

// WebReaderRequest represents a web reader request
type WebReaderRequest struct {
	URL          string `json:"url"`
	WithImages   bool   `json:"with_images,omitempty"`
	WithSummary  bool   `json:"with_summary,omitempty"`
	WithLinks    bool   `json:"with_links,omitempty"`
}

// WebReaderResponse represents web reader results
type WebReaderResponse struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Data    *WebReaderData `json:"data,omitempty"`
	Success bool   `json:"success"`
}

// WebReaderData represents parsed web page data
type WebReaderData struct {
	Title      string            `json:"title"`
	Content    string            `json:"content"`
	URL        string            `json:"url"`
	Images     []string          `json:"images,omitempty"`
	Links      []string          `json:"links,omitempty"`
	Summary    string            `json:"summary,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// TokenizerRequest represents a tokenization request
type TokenizerRequest struct {
	Text string `json:"text"`
	Model string `json:"model,omitempty"`
}

// TokenizerResponse represents tokenization results
type TokenizerResponse struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Data    *TokenizerData `json:"data,omitempty"`
	Success bool   `json:"success"`
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
func (s *ToolsService) WebSearch(query string, topK int) (*WebSearchResponse, error) {
	var result WebSearchResponse
	
	url := "https://api.z.ai/api/tools/web/search"
	
	request := WebSearchRequest{
		Query:  query,
		TopK:   topK,
		Stream: false,
	}
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+s.client.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &result, nil
}

// WebReader reads and parses content from a URL
func (s *ToolsService) WebReader(url string, withImages, withSummary bool) (*WebReaderResponse, error) {
	var result WebReaderResponse
	
	apiURL := "https://api.z.ai/api/tools/web/reader"
	
	request := WebReaderRequest{
		URL:         url,
		WithImages:  withImages,
		WithSummary: withSummary,
		WithLinks:   true,
	}
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+s.client.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &result, nil
}

// Tokenize counts tokens for a given text
func (s *ToolsService) Tokenize(text, model string) (*TokenizerResponse, error) {
	var result TokenizerResponse
	
	url := "https://api.z.ai/api/tools/tokenizer"
	
	request := TokenizerRequest{
		Text:  text,
		Model: model,
	}
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+s.client.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &result, nil
}
