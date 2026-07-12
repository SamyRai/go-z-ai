package client

import (
	"context"
	"fmt"
)

// ImagesService handles image generation.
type ImagesService struct {
	client *Client
}

// ImageGenerationRequest is the request body for image generation.
type ImageGenerationRequest struct {
	Model   string `json:"model"` // glm-image | cogview-4-250304 | cogview-4 | cogview-3-flash
	Prompt  string `json:"prompt"`
	Quality string `json:"quality,omitempty"` // hd (default) | standard
	Size    string `json:"size,omitempty"`    // default 1280x1280
	UserID  string `json:"user_id,omitempty"`
	// WatermarkEnabled controls the AI-generated watermark (both a visible
	// mark and an embedded digital one), which the API defaults to true. A
	// pointer so an explicit false survives — omitempty on a plain bool
	// would silently drop it and let the API's true default apply instead.
	WatermarkEnabled *bool `json:"watermark_enabled,omitempty"`
}

// ImageGenerationResponse is the response from a synchronous (or completed
// async) image generation request.
type ImageGenerationResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		URL string `json:"url"` // expires 30 days after generation
	} `json:"data"`
	ContentFilter []struct {
		Role  string `json:"role"`  // assistant | user | history
		Level int    `json:"level"` // 0 (most severe) - 3
	} `json:"content_filter,omitempty"`
}

// Generate creates an image synchronously, blocking until it's ready
// (~5-10s for standard quality, ~20s for hd).
func (s *ImagesService) Generate(ctx context.Context, req ImageGenerationRequest) (*ImageGenerationResponse, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	var resp ImageGenerationResponse
	if err := s.client.doRequest(ctx, "POST", "/images/generations", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}
	return &resp, nil
}

// GenerateAsync submits an image generation task and returns immediately;
// poll the result with Client.GetAsyncResult(resp.ID).
func (s *ImagesService) GenerateAsync(ctx context.Context, req ImageGenerationRequest) (*AsyncTaskResponse, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	var resp AsyncTaskResponse
	if err := s.client.doRequest(ctx, "POST", "/async/images/generations", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to submit async image generation: %w", err)
	}
	return &resp, nil
}
