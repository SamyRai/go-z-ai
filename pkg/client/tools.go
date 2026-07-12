package client

import (
	"context"
	"fmt"
)

// ToolsService wraps Z.AI's standalone tool endpoints — web search, web
// page reading, and tokenization — all under the general chat-completions
// base (Config.BaseURL), not a separate gateway. Endpoints and request/
// response shapes confirmed 2026-07-11 against docs.bigmodel.cn's live
// OpenAPI spec (https://docs.bigmodel.cn/openapi/openapi.json:
// POST /paas/v4/web_search, /paas/v4/reader, /paas/v4/tokenizer) and the
// official Python SDK source (api_resource/web_search/web_search.py,
// api_resource/web_reader/web_reader.py, both posting to the same relative
// paths under the default base URL). This replaced an earlier version of
// this file that hit an invented "ToolsBaseURL" + "/web/search" etc. — a
// path that appears nowhere in the real API and was never live-verified.
type ToolsService struct {
	client *Client
}

// Web search engine choices for WebSearchRequest.SearchEngine.
const (
	SearchEngineStd      = "search_std" // Z.AI's basic search engine
	SearchEnginePro      = "search_pro" // Z.AI's advanced search engine
	SearchEngineProSogou = "search_pro_sogou"
	SearchEngineProQuark = "search_pro_quark"
)

// Recency filters for WebSearchRequest.SearchRecencyFilter.
const (
	SearchRecencyOneDay   = "oneDay"
	SearchRecencyOneWeek  = "oneWeek"
	SearchRecencyOneMonth = "oneMonth"
	SearchRecencyOneYear  = "oneYear"
	SearchRecencyNoLimit  = "noLimit" // default
)

// Content sizes for WebSearchRequest.ContentSize.
const (
	SearchContentMedium = "medium" // summary-level content
	SearchContentHigh   = "high"   // maximal context, more detail
)

// WebSearchRequest performs a web search via Z.AI's LLM-optimized search
// engine. SearchQuery, SearchEngine, and SearchIntent are all required by
// the API — SearchIntent's zero value (false) is itself a valid explicit
// choice ("skip intent recognition"), so it's never omitted on the wire.
type WebSearchRequest struct {
	SearchQuery         string `json:"search_query"`    // required, max 70 chars
	SearchEngine        string `json:"search_engine"`   // required, one of the SearchEngine* consts
	SearchIntent        bool   `json:"search_intent"`   // required
	Count               int    `json:"count,omitempty"` // 1-50, default 10
	SearchDomainFilter  string `json:"search_domain_filter,omitempty"`
	SearchRecencyFilter string `json:"search_recency_filter,omitempty"` // one of the SearchRecency* consts
	ContentSize         string `json:"content_size,omitempty"`          // SearchContentMedium/High
	RequestID           string `json:"request_id,omitempty"`
	UserID              string `json:"user_id,omitempty"`
	IncludeImage        bool   `json:"include_image,omitempty"`
}

// WebSearchIntent is one recognized query intent in WebSearchResponse.SearchIntent.
type WebSearchIntent struct {
	Query    string `json:"query"`
	Intent   string `json:"intent"` // SEARCH_ALL, SEARCH_NONE, SEARCH_ALWAYS
	Keywords string `json:"keywords"`
}

// WebSearchResult is one item in WebSearchResponse.SearchResult.
type WebSearchResult struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	Link        string `json:"link"`
	Media       string `json:"media"` // site name
	Icon        string `json:"icon"`
	Refer       string `json:"refer"` // citation marker index
	PublishDate string `json:"publish_date"`
}

// WebSearchResponse is the result of ToolsService.WebSearch.
type WebSearchResponse struct {
	ID           string            `json:"id"`
	Created      int64             `json:"created"`
	RequestID    string            `json:"request_id"`
	SearchIntent []WebSearchIntent `json:"search_intent,omitempty"`
	SearchResult []WebSearchResult `json:"search_result,omitempty"`
}

// WebSearch performs a web search. req.SearchQuery and req.SearchEngine
// (one of the SearchEngine* constants) are required.
func (s *ToolsService) WebSearch(ctx context.Context, req WebSearchRequest) (*WebSearchResponse, error) {
	if req.SearchQuery == "" {
		return nil, fmt.Errorf("search_query is required")
	}
	if req.SearchEngine == "" {
		return nil, fmt.Errorf("search_engine is required")
	}

	var resp WebSearchResponse
	if err := s.client.doRequest(ctx, "POST", "/web_search", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	return &resp, nil
}

// WebReaderRequest reads and parses a URL's content. RetainImages is a
// pointer because the API defaults it to true — a plain bool with
// `omitempty` would silently drop an explicit false and fall back to the
// API's default instead of honoring it.
type WebReaderRequest struct {
	URL               string `json:"url"`               // required
	Timeout           int    `json:"timeout,omitempty"` // seconds, API default 20
	NoCache           bool   `json:"no_cache,omitempty"`
	ReturnFormat      string `json:"return_format,omitempty"` // e.g. "markdown" (API default), "text"
	RetainImages      *bool  `json:"retain_images,omitempty"` // API default true
	NoGFM             bool   `json:"no_gfm,omitempty"`
	KeepImgDataURL    bool   `json:"keep_img_data_url,omitempty"`
	WithImagesSummary bool   `json:"with_images_summary,omitempty"`
	WithLinksSummary  bool   `json:"with_links_summary,omitempty"`
}

// WebReaderMetadata is page metadata extracted by WebReader.
type WebReaderMetadata struct {
	Keywords        string `json:"keywords,omitempty"`
	Viewport        string `json:"viewport,omitempty"`
	Description     string `json:"description,omitempty"`
	FormatDetection string `json:"format-detection,omitempty"`
}

// WebReaderResult is the parsed page content in WebReaderResponse.
type WebReaderResult struct {
	Content     string             `json:"content"` // main body, with image/link markup
	Description string             `json:"description,omitempty"`
	Title       string             `json:"title"`
	URL         string             `json:"url"`
	Metadata    *WebReaderMetadata `json:"metadata,omitempty"`
}

// WebReaderResponse is the result of ToolsService.WebReader.
type WebReaderResponse struct {
	ID           string           `json:"id"`
	Created      int64            `json:"created"`
	RequestID    string           `json:"request_id"`
	Model        string           `json:"model"`
	ReaderResult *WebReaderResult `json:"reader_result,omitempty"`
}

// WebReader reads and parses content from a URL. req.URL is required.
func (s *ToolsService) WebReader(ctx context.Context, req WebReaderRequest) (*WebReaderResponse, error) {
	if req.URL == "" {
		return nil, fmt.Errorf("url is required")
	}

	var resp WebReaderResponse
	if err := s.client.doRequest(ctx, "POST", "/reader", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to read url: %w", err)
	}
	return &resp, nil
}

// TokenizerRequest counts tokens for a chat-completions-shaped request —
// the real API tokenizes a Model + Messages pair (identical shape to
// ChatRequest), not raw text.
type TokenizerRequest struct {
	Model     string    `json:"model"`    // required; one of the documented chat models
	Messages  []Message `json:"messages"` // required, at least one — must include a user/assistant message, not only system
	RequestID string    `json:"request_id,omitempty"`
	UserID    string    `json:"user_id,omitempty"`
}

// TokenizerUsage is the token breakdown in TokenizerResponse.
type TokenizerUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	VideoTokens  int `json:"video_tokens,omitempty"`
	ImageTokens  int `json:"image_tokens,omitempty"`
	TotalTokens  int `json:"total_tokens"`
}

// TokenizerResponse is the result of ToolsService.Tokenize.
type TokenizerResponse struct {
	ID        string          `json:"id"`
	Created   int64           `json:"created"`
	RequestID string          `json:"request_id"`
	Usage     *TokenizerUsage `json:"usage"`
}

// Tokenize counts tokens for a chat request without generating a
// completion — useful for cost/context-window estimation. req.Model and
// at least one req.Messages entry are required.
func (s *ToolsService) Tokenize(ctx context.Context, req TokenizerRequest) (*TokenizerResponse, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("at least one message is required")
	}

	var resp TokenizerResponse
	if err := s.client.doRequest(ctx, "POST", "/tokenizer", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to tokenize: %w", err)
	}
	return &resp, nil
}
