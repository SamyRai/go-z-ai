package client

import (
	"context"
	"fmt"
	"strings"
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

// GetTextModels returns all text-only models
func (s *ModelsService) GetTextModels(ctx context.Context) ([]ModelDetails, error) {
	return s.filterModels(ctx, func(m ModelDetails) bool { return isTextModel(m.ID) })
}

// GetVisionModels returns all vision-capable models
func (s *ModelsService) GetVisionModels(ctx context.Context) ([]ModelDetails, error) {
	return s.filterModels(ctx, func(m ModelDetails) bool { return isVisionModel(m.ID) })
}

// GetFreeModels returns all free models
func (s *ModelsService) GetFreeModels(ctx context.Context) ([]ModelDetails, error) {
	return s.filterModels(ctx, func(m ModelDetails) bool {
		return m.Pricing != nil && m.Pricing.Input == 0 && m.Pricing.Output == 0
	})
}

// RefreshCache clears and refreshes the models cache
func (s *ModelsService) RefreshCache(ctx context.Context) error {
	s.cacheMu.Lock()
	s.cache = nil
	s.cacheMu.Unlock()

	_, err := s.List(ctx)
	return err
}

// visionModelMarkers are substrings identifying vision-capable model IDs —
// the single source of truth isTextModel/isVisionModel both read, so the
// two categorizations can never drift out of sync with each other (they
// previously each hardcoded their own copy of this list).
var visionModelMarkers = []string{"glm-5v", "glm-4.6v", "glm-4.5v", "glm-ocr"}

func isVisionModel(modelID string) bool {
	for _, vm := range visionModelMarkers {
		if strings.Contains(modelID, vm) {
			return true
		}
	}
	return false
}

func isTextModel(modelID string) bool {
	return !isVisionModel(modelID)
}
