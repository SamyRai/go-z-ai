package provider

import (
	"fmt"
	"os"
	"sync"
)

// ProviderType represents different AI provider types
type ProviderType string

const (
	ProviderAnthropic ProviderType = "anthropic"
	ProviderOpenAI    ProviderType = "openai"
	ProviderZAI       ProviderType = "zai"
	ProviderCustom    ProviderType = "custom"
)

// ProviderConfig represents configuration for a specific provider
type ProviderConfig struct {
	Type        ProviderType `json:"type"`
	Name        string       `json:"name"`
	BaseURL     string       `json:"base_url"`
	APIKey      string       `json:"api_key"`
	Enabled     bool         `json:"enabled"`
	Model       string       `json:"model"`
	Headers     map[string]string `json:"headers,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

// ProviderManager manages multiple provider configurations
type ProviderManager struct {
	mu              sync.RWMutex
	providers       map[string]*ProviderConfig
	activeProvider  string
	configPath      string
}

// NewProviderManager creates a new provider manager
func NewProviderManager(configPath string) *ProviderManager {
	return &ProviderManager{
		providers:  make(map[string]*ProviderConfig),
		configPath: configPath,
	}
}

// AddProvider adds or updates a provider configuration
func (pm *ProviderManager) AddProvider(config *ProviderConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if config.Name == "" {
		return fmt.Errorf("provider name is required")
	}

	// Set default values based on provider type
	if err := pm.setDefaults(config); err != nil {
		return err
	}

	pm.providers[config.Name] = config

	// If this is the first provider or explicitly enabled, activate it
	if len(pm.providers) == 1 || config.Enabled {
		pm.activeProvider = config.Name
	}

	return nil
}

// RemoveProvider removes a provider configuration
func (pm *ProviderManager) RemoveProvider(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if name == pm.activeProvider {
		return fmt.Errorf("cannot remove active provider")
	}

	delete(pm.providers, name)
	return nil
}

// ActivateProvider switches to a specific provider
func (pm *ProviderManager) ActivateProvider(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.providers[name]; !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	pm.activeProvider = name

	// Release the lock before calling ApplyProviderEnvironment to avoid deadlock
	pm.mu.Unlock()
	err := pm.ApplyProviderEnvironment(name)
	pm.mu.Lock()

	return err
}

// GetActiveProvider returns the currently active provider configuration
func (pm *ProviderManager) GetActiveProvider() (*ProviderConfig, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.activeProvider == "" {
		return nil, fmt.Errorf("no active provider")
	}

	config, exists := pm.providers[pm.activeProvider]
	if !exists {
		return nil, fmt.Errorf("active provider %s not found", pm.activeProvider)
	}

	return config, nil
}

// ListProviders returns all configured providers
func (pm *ProviderManager) ListProviders() []*ProviderConfig {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	providers := make([]*ProviderConfig, 0, len(pm.providers))
	for _, config := range pm.providers {
		providers = append(providers, config)
	}
	return providers
}

// setDefaults sets default values based on provider type
func (pm *ProviderManager) setDefaults(config *ProviderConfig) error {
	switch config.Type {
	case ProviderAnthropic:
		if config.BaseURL == "" {
			config.BaseURL = "https://api.anthropic.com"
		}
		if config.Model == "" {
			config.Model = "claude-sonnet-4-20250514"
		}
		// Anthropic uses x-api-key header
		if config.Headers == nil {
			config.Headers = make(map[string]string)
		}
		config.Headers["x-api-key"] = config.APIKey

	case ProviderOpenAI:
		if config.BaseURL == "" {
			config.BaseURL = "https://api.openai.com/v1"
		}
		if config.Model == "" {
			config.Model = "gpt-4"
		}
		if config.Headers == nil {
			config.Headers = make(map[string]string)
		}
		config.Headers["Authorization"] = "Bearer " + config.APIKey

	case ProviderZAI:
		// Z.AI has different endpoints for different plans
		if config.BaseURL == "" {
			// Default to coding plan endpoint, will auto-detect
			config.BaseURL = "https://api.z.ai/api/coding/paas/v4"
		}
		if config.Model == "" {
			config.Model = "glm-4.7"
		}
		if config.Headers == nil {
			config.Headers = make(map[string]string)
		}
		config.Headers["Authorization"] = "Bearer " + config.APIKey

	case ProviderCustom:
		if config.BaseURL == "" {
			return fmt.Errorf("base_url is required for custom providers")
		}
		if config.Headers == nil {
			config.Headers = make(map[string]string)
		}
		if config.APIKey != "" {
			config.Headers["Authorization"] = "Bearer " + config.APIKey
		}

	default:
		return fmt.Errorf("unknown provider type: %s", config.Type)
	}

	// Set environment variables for this provider
	config.Environment = map[string]string{
		config.Type.String() + "_API_KEY":     config.APIKey,
		config.Type.String() + "_BASE_URL":   config.BaseURL,
		config.Type.String() + "_MODEL":       config.Model,
	}

	return nil
}

// ApplyProviderEnvironment sets environment variables for a provider
func (pm *ProviderManager) ApplyProviderEnvironment(name string) error {
	pm.mu.RLock()
	config, exists := pm.providers[name]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	// Set environment variables
	for key, value := range config.Environment {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", key, err)
		}
	}

	return nil
}

// GetProviderConfig gets a specific provider's configuration
func (pm *ProviderManager) GetProviderConfig(name string) (*ProviderConfig, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	config, exists := pm.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return config, nil
}

// UpdateProvider updates an existing provider configuration
func (pm *ProviderManager) UpdateProvider(name string, updates map[string]interface{}) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	config, exists := pm.providers[name]
	if !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	// Apply updates
	for key, value := range updates {
		switch key {
		case "base_url":
			if url, ok := value.(string); ok {
				config.BaseURL = url
			}
		case "api_key":
			if key, ok := value.(string); ok {
				config.APIKey = key
			}
		case "model":
			if model, ok := value.(string); ok {
				config.Model = model
			}
		case "enabled":
			if enabled, ok := value.(bool); ok {
				config.Enabled = enabled
			}
		}
	}

	// Reapply defaults and environment
	if err := pm.setDefaults(config); err != nil {
		return err
	}

	pm.providers[name] = config

	// If updating the active provider, reapply environment
	if name == pm.activeProvider {
		return pm.ApplyProviderEnvironment(name)
	}

	return nil
}

// DeactivateProvider disables the current provider
func (pm *ProviderManager) DeactivateProvider() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.activeProvider == "" {
		return fmt.Errorf("no active provider to deactivate")
	}

	// Clear environment variables
	activeConfig := pm.providers[pm.activeProvider]
	for key := range activeConfig.Environment {
		os.Unsetenv(key)
	}

	pm.activeProvider = ""
	return nil
}

// String returns the string representation of ProviderType
func (pt ProviderType) String() string {
	switch pt {
	case ProviderAnthropic:
		return "anthropic"
	case ProviderOpenAI:
		return "openai"
	case ProviderZAI:
		return "zai"
	case ProviderCustom:
		return "custom"
	default:
		return "unknown"
	}
}