package client

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// AccountType represents different Z.AI account types
type AccountType string

const (
	AccountTypePayAsYouGo AccountType = "pay_as_you_go"
	AccountTypeCodingPlan AccountType = "coding_plan"
	AccountTypeUnknown    AccountType = "unknown"
)

// AccountInfo represents detected account information
type DetectedAccount struct {
	Type        AccountType  `json:"type"`
	BaseURL     string       `json:"base_url"`
	Working     bool         `json:"working"`
	Models      []string     `json:"available_models"`
	UsageLimits *UsageLimits `json:"usage_limits,omitempty"`
}

// UsageLimits represents subscription usage limits
type UsageLimits struct {
	HourlyPromptLimit int    `json:"hourly_prompt_limit"`
	WeeklyPromptLimit int    `json:"weekly_prompt_limit"`
	HourlyWindowReset string `json:"hourly_window_reset"`
	WeeklyReset       string `json:"weekly_reset"`
	CurrentUsage      int    `json:"current_usage"`
	RemainingQuota    int    `json:"remaining_quota"`
}

// DetectionService handles account type detection
type DetectionService struct {
	client *Client
	mu     sync.RWMutex
	cache  *DetectedAccount
}

// NewDetectionService creates a new detection service
func NewDetectionService(client *Client) *DetectionService {
	return &DetectionService{
		client: client,
	}
}

// DetectAccountType detects the account type by testing different endpoints
func (s *DetectionService) DetectAccountType(ctx context.Context) (*DetectedAccount, error) {
	// Try to use cached result
	s.mu.RLock()
	if s.cache != nil {
		defer s.mu.RUnlock()
		return s.cache, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Test Coding Plan endpoint
	codingResult := s.testEndpoint(ctx, CodingBaseURL)
	if codingResult.Working {
		account := &DetectedAccount{
			Type:        AccountTypeCodingPlan,
			BaseURL:     CodingBaseURL,
			Working:     true,
			Models:      []string{"glm-5.2", "glm-5-turbo", "glm-4.7"},
			UsageLimits: s.detectCodingPlanLimits(),
		}
		s.cache = account
		return account, nil
	}

	// Test Pay-as-you-go endpoint
	paasResult := s.testEndpoint(ctx, ProdBaseURL)
	if paasResult.Working {
		account := &DetectedAccount{
			Type:        AccountTypePayAsYouGo,
			BaseURL:     ProdBaseURL,
			Working:     true,
			Models:      []string{}, // Will be populated
			UsageLimits: nil,
		}
		s.cache = account
		return account, nil
	}

	return nil, fmt.Errorf("unable to detect account type - both endpoints failed")
}

func (s *DetectionService) testEndpoint(ctx context.Context, baseURL string) *EndpointTest {
	result := &EndpointTest{
		BaseURL: baseURL,
	}

	// Create a temporary client with the test endpoint URL
	tempClient, err := NewClient(Config{
		APIKey:  s.client.config.APIKey,
		BaseURL: baseURL,
	})
	if err != nil {
		result.Error = err
		return result
	}

	// Try to make a request
	_, err = tempClient.Chat().Create(ctx, ChatRequest{
		Model:       "glm-4.7",
		Messages:    []Message{{Role: "user", Content: "test"}},
		MaxTokens:   1,
		Temperature: 0.7,
		TopP:        0.95,
	})

	if err != nil {
		// Check if error indicates API is working but has other issues
		errMsg := err.Error()
		if strings.Contains(errMsg, "429") || strings.Contains(errMsg, "1113") || strings.Contains(errMsg, "rate limit") {
			// 429 rate limit means API is accessible
			result.Working = true
			result.Error = err
			return result
		}
		result.Error = err
		return result
	}

	result.Working = true
	return result
}

type EndpointTest struct {
	BaseURL    string
	Working    bool
	StatusCode int
	Error      error
}

func (s *DetectionService) detectCodingPlanLimits() *UsageLimits {
	// Z.AI doesn't provide API endpoints to get actual usage limits
	// These are general reference values from documentation
	// Your actual limits may vary based on your specific plan
	return &UsageLimits{
		HourlyPromptLimit: 0, // Unknown - would need API endpoint
		WeeklyPromptLimit: 0, // Unknown - would need API endpoint
		HourlyWindowReset: "5-hours (rolling window)",
		WeeklyReset:       "7-days (estimated)",
		CurrentUsage:      0,
		RemainingQuota:    0,
	}
}

// GetAccountInfo returns detected account information
func (s *DetectionService) GetAccountInfo(ctx context.Context) (*DetectedAccount, error) {
	account, err := s.DetectAccountType(ctx)
	if err != nil {
		return nil, err
	}
	return account, nil
}
