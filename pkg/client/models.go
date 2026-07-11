package client

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// ModelsService handles model-related operations
type ModelsService struct {
	client    *Client
	cache     *ModelsInfo
	cacheMu   sync.RWMutex
	cacheOnce sync.Once
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

// GetTextModels returns all text-only models
func (s *ModelsService) GetTextModels(ctx context.Context) ([]ModelDetails, error) {
	models, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	var textModels []ModelDetails
	for _, model := range models.Models {
		if isTextModel(model.ID) {
			textModels = append(textModels, model)
		}
	}

	return textModels, nil
}

// GetVisionModels returns all vision-capable models
func (s *ModelsService) GetVisionModels(ctx context.Context) ([]ModelDetails, error) {
	models, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	var visionModels []ModelDetails
	for _, model := range models.Models {
		if isVisionModel(model.ID) {
			visionModels = append(visionModels, model)
		}
	}

	return visionModels, nil
}

// GetFreeModels returns all free models
func (s *ModelsService) GetFreeModels(ctx context.Context) ([]ModelDetails, error) {
	models, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	var freeModels []ModelDetails
	for _, model := range models.Models {
		if model.Pricing != nil && model.Pricing.Input == 0 && model.Pricing.Output == 0 {
			freeModels = append(freeModels, model)
		}
	}

	return freeModels, nil
}

// RefreshCache clears and refreshes the models cache
func (s *ModelsService) RefreshCache(ctx context.Context) error {
	s.cacheMu.Lock()
	s.cache = nil
	s.cacheMu.Unlock()

	_, err := s.List(ctx)
	return err
}

// Helper functions to categorize models
func isTextModel(modelID string) bool {
	visionModels := []string{"glm-5v", "glm-4.6v", "glm-4.5v", "glm-ocr"}
	for _, vm := range visionModels {
		if strings.Contains(modelID, vm) {
			return false
		}
	}
	return true
}

func isVisionModel(modelID string) bool {
	visionModels := []string{"glm-5v", "glm-4.6v", "glm-4.5v", "glm-ocr"}
	for _, vm := range visionModels {
		if strings.Contains(modelID, vm) {
			return true
		}
	}
	return false
}
