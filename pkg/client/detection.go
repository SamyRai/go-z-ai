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

// DetectAccountType detects the account type by testing different endpoints.
// The probe hosts follow Config.Region: a RegionChina client tests the
// open.bigmodel.cn coding/paas endpoints, RegionGlobal (the default) tests
// api.z.ai — so a China-issued key is classified against its own platform
// rather than mis-failing auth on the global host.
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

	codingURL, paasURL := s.probeURLs()
	// Test Coding Plan endpoint
	codingResult := s.testEndpoint(ctx, codingURL)
	if codingResult.Working {
		account := &DetectedAccount{
			Type:        AccountTypeCodingPlan,
			BaseURL:     codingURL,
			Working:     true,
			Models:      []string{"glm-5.2", "glm-5-turbo", "glm-4.7"},
			UsageLimits: s.detectCodingPlanLimits(),
		}
		s.cache = account
		return account, nil
	}

	// Test Pay-as-you-go endpoint
	paasResult := s.testEndpoint(ctx, paasURL)
	if paasResult.Working {
		account := &DetectedAccount{
			Type:        AccountTypePayAsYouGo,
			BaseURL:     paasURL,
			Working:     true,
			Models:      []string{}, // Will be populated
			UsageLimits: nil,
		}
		s.cache = account
		return account, nil
	}

	return nil, fmt.Errorf("unable to detect account type - both endpoints failed")
}

// probeURLs returns the coding-plan and pay-as-you-go base URLs to probe,
// selected by Config.Region. The China mirror paths mirror api.z.ai's layout
// (live-verified for /models and /chat/completions — see BigModelBaseURL).
func (s *DetectionService) probeURLs() (coding, paas string) {
	if s.client.config.Region == RegionChina {
		return ChinaCodingBaseURL, ChinaProdBaseURL
	}
	return CodingBaseURL, ProdBaseURL
}

func (s *DetectionService) testEndpoint(ctx context.Context, baseURL string) *EndpointTest {
	result := &EndpointTest{
		BaseURL: baseURL,
	}

	// Create a temporary client pointed at the probe URL. It inherits the
	// parent's HTTPClient so a test (or a caller supplying a custom transport)
	// can intercept the probe instead of hitting the real network; only the
	// BaseURL is overridden. MaxRetries=-1 disables retries — a probe that
	// fails shouldn't burn 3 backoff rounds before detection falls through to
	// the next endpoint. ChinaAPIKey/Region/etc. are irrelevant for this
	// one-shot probe against an explicit URL.
	tempClient, err := NewClient(Config{
		APIKey:     s.client.config.APIKey,
		BaseURL:    baseURL,
		HTTPClient: s.client.httpClient,
		MaxRetries: -1,
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
