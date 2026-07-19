package client

import (
	"context"
	"fmt"
	"sync"
)

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
