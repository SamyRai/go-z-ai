package client

import (
	"context"
	"errors"
	"fmt"
)

// Video generation models.
const (
	ModelCogVideoX3     = "cogvideox-3" // text/image-to-video up to 4K
	ModelViduQ1Text     = "viduq1-text"
	ModelViduQ1Image    = "viduq1-image"
	ModelViduQ1StartEnd = "viduq1-start-end"
	ModelVidu2Image     = "vidu2-image"
	ModelVidu2StartEnd  = "vidu2-start-end"
	ModelVidu2Reference = "vidu2-reference"
)

// VideoModels lists the video generation models.
var VideoModels = []string{
	ModelCogVideoX3, ModelViduQ1Text, ModelViduQ1Image, ModelViduQ1StartEnd,
	ModelVidu2Image, ModelVidu2StartEnd, ModelVidu2Reference,
}

// VideosService handles video generation. Video generation is always
// asynchronous — Generate returns a task to poll via Client.GetAsyncResult.
type VideosService struct {
	client *Client
}

// VideoGenerationRequest is the union of fields across Z.AI's video models
// (cogvideox-3, viduq1-text, viduq1-image/vidu2-image,
// viduq1-start-end/vidu2-start-end, vidu2-reference). The API ignores
// fields that don't apply to the chosen Model, so one struct covers all of
// them rather than five near-identical request types.
type VideoGenerationRequest struct {
	Model     string `json:"model"`
	RequestID string `json:"request_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`

	Prompt   string   `json:"prompt,omitempty"`    // <=512 chars
	ImageURL []string `json:"image_url,omitempty"` // 1+ images depending on model; single-image models may pass one element

	Size              string `json:"size,omitempty"`
	AspectRatio       string `json:"aspect_ratio,omitempty"`       // 16:9 | 9:16 | 1:1
	Duration          int    `json:"duration,omitempty"`           // seconds; valid values vary by model
	FPS               int    `json:"fps,omitempty"`                // cogvideox-3 only: 30 | 60
	Style             string `json:"style,omitempty"`              // viduq1-text only: general | anime
	Quality           string `json:"quality,omitempty"`            // cogvideox-3 only: speed | quality
	MovementAmplitude string `json:"movement_amplitude,omitempty"` // auto | small | medium | large
	WithAudio         bool   `json:"with_audio,omitempty"`
	// OffPeak queues the task for off-peak processing at a lower price.
	OffPeak bool `json:"off_peak,omitempty"`
	// WatermarkEnabled controls the AI-generated watermark, which the API
	// defaults to true. A pointer so an explicit false survives — see
	// ImageGenerationRequest.WatermarkEnabled for why (omitempty would
	// silently drop a false zero value and let the API's true default apply).
	WatermarkEnabled *bool `json:"watermark_enabled,omitempty"`
}

// Generate submits a video generation task and returns immediately; poll
// the result with Client.GetAsyncResult(resp.ID).
func (s *VideosService) Generate(ctx context.Context, req VideoGenerationRequest) (*AsyncTaskResponse, error) {
	if req.Model == "" {
		return nil, errors.New("model is required")
	}
	if req.Prompt == "" && len(req.ImageURL) == 0 {
		return nil, fmt.Errorf("prompt or image_url is required")
	}

	var resp AsyncTaskResponse
	if err := s.client.doRequest(ctx, "POST", "/videos/generations", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to submit video generation: %w", err)
	}
	return &resp, nil
}
