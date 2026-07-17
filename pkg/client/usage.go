package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// UsageService handles client-side usage tracking and balance checking
type UsageService struct {
	client  *Client
	tracker *UsageTracker
	mu      sync.RWMutex
}

// UsageTracker provides client-side usage tracking
type UsageTracker struct {
	requests    map[string]int     // model -> request count
	tokens      map[string]int64   // model -> total tokens
	costs       map[string]float64 // model -> total cost
	lastUpdated time.Time
	mu          sync.RWMutex
}

// NewUsageTracker creates a new usage tracker
func NewUsageTracker() *UsageTracker {
	return &UsageTracker{
		requests:    make(map[string]int),
		tokens:      make(map[string]int64),
		costs:       make(map[string]float64),
		lastUpdated: time.Now(),
	}
}

// TrackRequest logs an API request for usage tracking
func (t *UsageTracker) TrackRequest(model string, promptTokens, completionTokens int, cost float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.requests[model]++
	t.tokens[model] += int64(promptTokens + completionTokens)
	t.costs[model] += cost
	t.lastUpdated = time.Now()
}

// GetSummary returns current usage summary
func (t *UsageTracker) GetSummary() map[string]any {
	t.mu.RLock()
	defer t.mu.RUnlock()

	totalRequests := 0
	totalTokens := int64(0)
	totalCost := 0.0

	for model, count := range t.requests {
		totalRequests += count
		totalTokens += t.tokens[model]
		totalCost += t.costs[model]
	}

	return map[string]any{
		"total_requests":  totalRequests,
		"total_tokens":    totalTokens,
		"total_cost":      totalCost,
		"by_model":        t.requests,
		"tokens_by_model": t.tokens,
		"costs_by_model":  t.costs,
		"last_updated":    t.lastUpdated,
	}
}

// TestBalance checks if account has sufficient balance
func (s *UsageService) TestBalance(ctx context.Context) error {
	s.mu.Lock()
	if s.tracker == nil {
		s.tracker = NewUsageTracker()
	}
	s.mu.Unlock()

	// Make a minimal API call to test balance
	testReq := ChatRequest{
		Model:       "glm-4.5",
		Messages:    []Message{{Role: "user", Content: "test"}},
		MaxTokens:   1,
		Temperature: 0.7,
		TopP:        0.95,
	}

	// Return the underlying error unchanged (it carries a structured *APIError
	// with the business Code and HTTP status). GetAccountStatus classifies it
	// via errors.As — earlier this method flattened a 1113 balance error into a
	// plain string with fmt.Errorf, which dropped the markers GetAccountStatus
	// then tried to match, making its insufficient-balance branch unreachable.
	_, err := s.client.Chat().Create(ctx, testReq)
	return err
}

// GetClientSideUsage returns client-side tracked usage information
func (s *UsageService) GetClientSideUsage() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.tracker == nil {
		return map[string]any{
			"message": "No usage tracked yet. Make API requests to enable tracking.",
		}
	}

	return s.tracker.GetSummary()
}

// GetModelPricing returns pricing information for a specific model
func (s *UsageService) GetModelPricing(ctx context.Context, model string) (map[string]float64, error) {
	models, err := s.client.Models().List(ctx)
	if err != nil {
		return nil, err
	}

	for _, m := range models.Models {
		if m.ID == model && m.Pricing != nil {
			return map[string]float64{
				"input":          m.Pricing.Input,
				"output":         m.Pricing.Output,
				"cached":         m.Pricing.Cached,
				"cached_storage": m.Pricing.CacheStore,
			}, nil
		}
	}

	return nil, fmt.Errorf("model not found: %s", model)
}

// CalculateRequestCost calculates the cost of a request based on model pricing
func (s *UsageService) CalculateRequestCost(ctx context.Context, model string, promptTokens, completionTokens int) (float64, error) {
	pricing, err := s.GetModelPricing(ctx, model)
	if err != nil {
		return 0, err
	}

	inputCost := (float64(promptTokens) / 1_000_000) * pricing["input"]
	outputCost := (float64(completionTokens) / 1_000_000) * pricing["output"]

	return inputCost + outputCost, nil
}

// AccountStatus represents the current account status
type AccountStatus struct {
	APIAccessible bool      `json:"api_accessible"`
	HasBalance    bool      `json:"has_balance"`
	LastChecked   time.Time `json:"last_checked"`
	Message       string    `json:"message"`
	WebDashboard  string    `json:"web_dashboard"`
}

// GetAccountStatus checks the current account status
func (s *UsageService) GetAccountStatus(ctx context.Context) (*AccountStatus, error) {
	status := &AccountStatus{
		LastChecked:  time.Now(),
		WebDashboard: "https://z.ai",
	}

	// Test API accessibility and balance. Classify the failure from the
	// structured *APIError (business Code + HTTP status) via errors.As rather
	// than string-matching a message we don't fully control.
	err := s.TestBalance(ctx)
	if err != nil {
		// Any error means we can't confirm balance.
		status.APIAccessible = false
		status.HasBalance = false

		if apiErr, ok := errors.AsType[*APIError](err); ok {
			switch {
			case isBalanceCode(apiErr.Code):
				// Auth succeeded and the request reached billing — the key
				// works, there's just no balance/quota to spend.
				status.APIAccessible = true
				status.Message = "API accessible but insufficient balance - please recharge at https://z.ai"
			case apiErr.HTTPStatus == http.StatusUnauthorized:
				status.Message = "API key authentication failed - check your API key"
			case apiErr.HTTPStatus == http.StatusTooManyRequests:
				status.APIAccessible = true
				status.Message = "API accessible but rate limited - try again later"
			default:
				status.Message = extractCleanError(err.Error())
			}
			return status, nil
		}

		// Non-API error (network, timeout, etc.).
		status.Message = extractCleanError(err.Error())
		return status, nil
	}

	status.APIAccessible = true
	status.HasBalance = true
	status.Message = "Account is healthy and has balance"
	return status, nil
}

// isBalanceCode reports whether code is one of Z.AI's "authenticated but out of
// balance/quota" business codes — the account key is valid, there's just
// nothing to spend, so callers should treat the API as accessible.
func isBalanceCode(code int) bool {
	switch code {
	case ErrCodeInsufficientBalance, ErrCodeHourlyLimitNoBalance, ErrCodeWeeklyLimitNoBalance:
		return true
	default:
		return false
	}
}

// GetWebDashboardURL returns the URL for the web dashboard
func (s *UsageService) GetWebDashboardURL() string {
	return "https://z.ai"
}

// extractCleanError removes redundant error messages
func extractCleanError(errMsg string) string {
	// Remove common prefixes step by step
	cleaned := errMsg

	// Remove "failed to create chat completion: "
	cleaned = strings.TrimPrefix(cleaned, "failed to create chat completion: ")

	// Extract JSON message if present
	if strings.Contains(cleaned, `"message":`) {
		// Find the message value
		msgStart := indexOfString(cleaned, `"message":"`) + 11 // skip "message":"
		if msgStart > 10 && msgStart < len(cleaned) {
			msgEnd := msgStart
			for i := msgStart; i < len(cleaned); i++ {
				if cleaned[i] == '"' && (i == 0 || cleaned[i-1] != '\\') {
					msgEnd = i
					break
				}
			}
			if msgEnd > msgStart {
				return cleaned[msgStart:msgEnd]
			}
		}
	}

	return cleaned
}

// indexOfString finds the index of a substring
func indexOfString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
