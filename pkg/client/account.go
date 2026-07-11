package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// AccountService handles account and profile information
type AccountService struct {
	client *Client
}

// AccountInfoResponse represents account information response
type AccountInfoResponse struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Data    *AccountData `json:"data,omitempty"`
	Success bool   `json:"success"`
}

// AccountData represents detailed account information
type AccountData struct {
	UserID       string    `json:"user_id,omitempty"`
	Email        string    `json:"email,omitempty"`
	AccountType  string    `json:"account_type,omitempty"`
	Status       string    `json:"status,omitempty"`
	Balance      float64   `json:"balance,omitempty"`
	Credit       float64   `json:"credit,omitempty"`
	Currency     string    `json:"currency,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	Verified     bool      `json:"verified,omitempty"`
}

// AccountStatusResponse represents account status response
type AccountStatusResponse struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Data    *AccountStatusData `json:"data,omitempty"`
	Success bool   `json:"success"`
}

// AccountStatusData represents account status data
type AccountStatusData struct {
	AccountID    string    `json:"account_id,omitempty"`
	Status       string    `json:"status,omitempty"`
	Plan         string    `json:"plan,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	HasBalance   bool      `json:"has_balance,omitempty"`
	QuotaStatus  string    `json:"quota_status,omitempty"`
}

// NewAccountService creates a new account service
func NewAccountService(client *Client) *AccountService {
	return &AccountService{client: client}
}

// GetAccountInfo retrieves account information
func (s *AccountService) GetAccountInfo() (*AccountInfoResponse, error) {
	var result AccountInfoResponse

	// Use business API base URL
	url := "https://api.z.ai/api/biz/account/info"
	
	// Create a temporary HTTP client for business API
	tempClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.client.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := tempClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetAccountStatus retrieves account status
func (s *AccountService) GetAccountStatus() (*AccountStatusResponse, error) {
	var result AccountStatusResponse

	// Use business API base URL
	url := "https://api.z.ai/api/biz/account/status"
	
	// Create a temporary HTTP client for business API
	tempClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.client.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := tempClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
