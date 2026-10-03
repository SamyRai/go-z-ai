package client

import (
	"context"
	"fmt"
	"sync"
)

// ModelsInfo represents available models information
type ModelsInfo struct {
	Models []ModelDetails `json:"data"`
}

// ModelDetails describes one model. The /models endpoint returns only the
// OpenAI-bare {id, object, created, owned_by} shape; ModelsService.List
// fills the remaining fields from the curated catalog (models_catalog.go),
// with live API values always winning when the API does send them.
type ModelDetails struct {
	ID          string   `json:"id"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	OwnedBy     string   `json:"owned_by,omitempty"`
	Created     int64    `json:"created,omitempty"` // release time, Unix seconds
	ContextSize int      `json:"max_context,omitempty"`
	MaxOutput   int      `json:"max_output,omitempty"`
	Pricing     *Pricing `json:"pricing,omitempty"`
	// Family groups related variants ("GLM-5", "GLM-4"); Tier is a short
	// label ("flagship", "flash", "vision", ...).
	Family string `json:"family,omitempty"`
	Tier   string `json:"tier,omitempty"`
	// Capabilities is the set of Cap* codes the model supports; empty means
	// unknown.
	Capabilities []string `json:"capabilities,omitempty"`
	// ReasoningEfforts lists the ChatRequest.ReasoningEffort levels the
	// model accepts; empty means none or unknown.
	ReasoningEfforts []string `json:"reasoning_efforts,omitempty"`
}

// Pricing is a model's token pricing (Unit is "USD/1M" for catalog values).
type Pricing struct {
	Input  float64 `json:"prompt"`
	Output float64 `json:"completion"`
	Cached float64 `json:"cached_prompt,omitempty"` // cached-input rate
	Unit   string  `json:"unit,omitempty"`
}

// Cost returns the cost of usage at these rates (Unit "USD/1M"), billing
// cached prompt tokens at the cached rate when one is known.
func (p Pricing) Cost(u Usage) float64 {
	prompt, cached := float64(u.PromptTokens), 0.0
	if u.PromptTokensDetails != nil && p.Cached > 0 {
		cached = float64(u.PromptTokensDetails.CachedTokens)
		prompt -= cached
	}
	return (prompt*p.Input + cached*p.Cached + float64(u.CompletionTokens)*p.Output) / 1_000_000
}

// ModelsService handles model-related operations
type ModelsService struct {
	client  *Client
	cache   *ModelsInfo
	cacheMu sync.RWMutex
}

// List returns all available models
func (s *ModelsService) List(ctx context.Context) (*ModelsInfo, error) {
	// Try to get from cache first
	s.cacheMu.RLock()
	if s.cache != nil {
		defer s.cacheMu.RUnlock()
		return s.cache, nil
	}
	s.cacheMu.RUnlock()

	// Fetch fresh data
	var response struct {
		Object string         `json:"object"`
		Data   []ModelDetails `json:"data"`
	}

	err := s.client.doRequest(ctx, "GET", "/models", nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	// Overlay catalog metadata (context, pricing, capabilities, name,
	// description) onto each model the API returned. The /models endpoint
	// currently sends only the OpenAI-bare {id, object, created, owned_by}
	// shape, so without this every Context/Pricing cell renders as `-`/0.
	// See models_catalog.go for the source of truth and the refresh notes.
	for i := range response.Data {
		response.Data[i] = enrichModel(response.Data[i])
	}

	modelsInfo := &ModelsInfo{
		Models: response.Data,
	}

	// Cache the result
	s.cacheMu.Lock()
	s.cache = modelsInfo
	s.cacheMu.Unlock()

	return modelsInfo, nil
}

// Get returns details for a specific model
func (s *ModelsService) Get(ctx context.Context, modelID string) (*ModelDetails, error) {
	if modelID == "" {
		return nil, fmt.Errorf("model ID is required")
	}

	models, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, model := range models.Models {
		if model.ID == modelID {
			return &model, nil
		}
	}

	return nil, fmt.Errorf("model not found: %s", modelID)
}

// filterModels lists all models and returns those matching keep — shared by
// GetTextModels/GetVisionModels/GetFreeModels so the list-then-filter shape
// lives in one place.
func (s *ModelsService) filterModels(ctx context.Context, keep func(ModelDetails) bool) ([]ModelDetails, error) {
	models, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	var result []ModelDetails
	for _, model := range models.Models {
		if keep(model) {
			result = append(result, model)
		}
	}
	return result, nil
}

// GetTextModels returns all text-capable models (every chat model advertises
// the "text" capability in the catalog; vision models do too, since they also
// chat). If you want text-ONLY models (excluding vision), filter on
// HasCapability("text") && !HasCapability("vision") at the callsite.
func (s *ModelsService) GetTextModels(ctx context.Context) ([]ModelDetails, error) {
	return s.filterModels(ctx, func(m ModelDetails) bool { return m.HasCapability(CapText) })
}

// GetVisionModels returns all vision-capable models.
func (s *ModelsService) GetVisionModels(ctx context.Context) ([]ModelDetails, error) {
	return s.filterModels(ctx, func(m ModelDetails) bool { return m.HasCapability(CapVision) })
}

// GetFreeModels returns all genuinely-free models — those with a non-nil
// Pricing whose input and output rates are both zero. See ModelDetails.IsFree
// for why nil Pricing is treated as "unknown", not "free".
func (s *ModelsService) GetFreeModels(ctx context.Context) ([]ModelDetails, error) {
	return s.filterModels(ctx, func(m ModelDetails) bool { return m.IsFree() })
}

// RefreshCache clears and refreshes the models cache
func (s *ModelsService) RefreshCache(ctx context.Context) error {
	s.cacheMu.Lock()
	s.cache = nil
	s.cacheMu.Unlock()

	_, err := s.List(ctx)
	return err
}
