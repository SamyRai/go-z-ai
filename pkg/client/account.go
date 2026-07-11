package client

import (
	"context"
	"fmt"
	"time"
)

// AccountService handles account and profile information
type AccountService struct {
	client *Client
}

// AccountInfoResponse represents account information response
type AccountInfoResponse struct {
	Code    int          `json:"code"`
	Msg     string       `json:"msg"`
	Data    *AccountData `json:"data,omitempty"`
	Success bool         `json:"success"`
}

// AccountData represents detailed account information
type AccountData struct {
	UserID      string    `json:"user_id,omitempty"`
	Email       string    `json:"email,omitempty"`
	AccountType string    `json:"account_type,omitempty"`
	Status      string    `json:"status,omitempty"`
	Balance     float64   `json:"balance,omitempty"`
	Credit      float64   `json:"credit,omitempty"`
	Currency    string    `json:"currency,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	Verified    bool      `json:"verified,omitempty"`
}

// AccountStatusResponse represents account status response
type AccountStatusResponse struct {
	Code    int                `json:"code"`
	Msg     string             `json:"msg"`
	Data    *AccountStatusData `json:"data,omitempty"`
	Success bool               `json:"success"`
}

// AccountStatusData represents account status data
type AccountStatusData struct {
	AccountID   string    `json:"account_id,omitempty"`
	Status      string    `json:"status,omitempty"`
	Plan        string    `json:"plan,omitempty"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	HasBalance  bool      `json:"has_balance,omitempty"`
	QuotaStatus string    `json:"quota_status,omitempty"`
}

// NewAccountService creates a new account service
func NewAccountService(client *Client) *AccountService {
	return &AccountService{client: client}
}

// GetAccountInfo retrieves account information
func (s *AccountService) GetAccountInfo(ctx context.Context) (*AccountInfoResponse, error) {
	var result AccountInfoResponse
	if err := s.client.doRequestBase(ctx, BizBaseURL, "GET", "/account/info", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}
	return &result, nil
}

// GetAccountStatus retrieves account status
func (s *AccountService) GetAccountStatus(ctx context.Context) (*AccountStatusResponse, error) {
	var result AccountStatusResponse
	if err := s.client.doRequestBase(ctx, BizBaseURL, "GET", "/account/status", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get account status: %w", err)
	}
	return &result, nil
}
