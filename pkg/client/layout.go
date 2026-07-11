package client

import (
	"context"
	"fmt"
)

// LayoutService handles layout parsing (OCR / document-to-markdown).
type LayoutService struct {
	client *Client
}

// LayoutParsingRequest is the request body for layout parsing. Model is
// always "glm-ocr". File is an image or PDF, as a URL or base64 string.
type LayoutParsingRequest struct {
	Model                   string `json:"model"`
	File                    string `json:"file"`
	ReturnCropImages        bool   `json:"return_crop_images,omitempty"`
	NeedLayoutVisualization bool   `json:"need_layout_visualization,omitempty"`
	StartPageID             int    `json:"start_page_id,omitempty"` // PDFs only
	EndPageID               int    `json:"end_page_id,omitempty"`   // PDFs only
	RequestID               string `json:"request_id,omitempty"`
	UserID                  string `json:"user_id,omitempty"`
}

// LayoutParsingResponse is the parsed document result.
type LayoutParsingResponse struct {
	ID                  string   `json:"id"`
	Created             int64    `json:"created"`
	Model               string   `json:"model"`
	MDResults           string   `json:"md_results"` // recognized content as Markdown
	LayoutDetails       []any    `json:"layout_details,omitempty"`
	LayoutVisualization []string `json:"layout_visualization,omitempty"`
	DataInfo            struct {
		NumPages int `json:"num_pages"`
	} `json:"data_info"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}

const layoutParsingModel = "glm-ocr"

// Parse recognizes an image or PDF's layout, returning the content as
// Markdown. req.Model is set to "glm-ocr" automatically if empty.
func (s *LayoutService) Parse(ctx context.Context, req LayoutParsingRequest) (*LayoutParsingResponse, error) {
	if req.File == "" {
		return nil, fmt.Errorf("file is required")
	}
	if req.Model == "" {
		req.Model = layoutParsingModel
	}

	var resp LayoutParsingResponse
	if err := s.client.doRequest(ctx, "POST", "/layout_parsing", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse layout: %w", err)
	}
	return &resp, nil
}
