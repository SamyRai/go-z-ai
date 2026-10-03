package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// BalanceProbeModel is the model CheckAccountStatus bills a one-token probe
// against for pay-as-you-go keys: the cheapest paid text model, so the probe
// exercises billing (a free model would succeed with an empty balance) at a
// negligible cost.
const BalanceProbeModel = "glm-4.7-flashx"

// AccountStatus reports whether an API key authenticates and can currently
// spend.
type AccountStatus struct {
	Account       *DetectedAccount `json:"account,omitempty"`
	APIAccessible bool             `json:"api_accessible"`
	// HasBalance is true when the key can spend right now: a pay-as-you-go
	// balance, or a coding plan with every token window below its limit.
	HasBalance  bool      `json:"has_balance"`
	Message     string    `json:"message"`
	LastChecked time.Time `json:"last_checked"`
}

// CheckAccountStatus reports whether the client's key works and can spend.
//
// Coding-plan keys are checked for free through the quota endpoint. Z.AI
// exposes no balance API for pay-as-you-go keys, so for those this sends one
// minimal billed completion (one output token on BalanceProbeModel, thinking
// off) and classifies the outcome. Call it on demand — never on a timer.
func (s *DetectionService) CheckAccountStatus(ctx context.Context) (*AccountStatus, error) {
	status := &AccountStatus{LastChecked: time.Now()}
	account, err := s.DetectAccountType(ctx)
	if err != nil {
		return nil, err
	}
	status.Account = account

	if account.Type == AccountTypeCodingPlan {
		quota, err := s.client.Quota().GetQuotaLimit(ctx)
		if err != nil {
			return nil, err
		}
		status.APIAccessible = true
		status.HasBalance = !quota.Data.Exhausted()
		status.Message = fmt.Sprintf("GLM Coding Plan (%s) active", account.Level)
		if !status.HasBalance {
			status.Message += " — a usage window is exhausted; wait for its reset"
		}
		return status, nil
	}

	probe := ChatRequest{
		Model:     BalanceProbeModel,
		Messages:  []Message{{Role: "user", Content: "ping"}},
		MaxTokens: 1,
		Thinking:  &ThinkingConfig{Type: ThinkingDisabled},
	}
	_, err = s.client.Chat().Create(ctx, probe)
	switch apiErr, isAPIErr := errors.AsType[*APIError](err); {
	case err == nil:
		status.APIAccessible, status.HasBalance = true, true
		status.Message = "API key works and the account has balance"
	case !isAPIErr:
		return nil, err
	case apiErr.IsBalanceError():
		status.APIAccessible = true
		status.Message = "API key works but the account has no balance — recharge at " + account.Region.ConsoleURL()
	case apiErr.HTTPStatus == http.StatusTooManyRequests:
		status.APIAccessible, status.HasBalance = true, true
		status.Message = "API key works but is rate limited — try again later"
	case apiErr.IsAuthError() || apiErr.HTTPStatus == http.StatusUnauthorized:
		status.Message = "API key authentication failed — check the key"
	default:
		status.Message = apiErr.Error()
	}
	return status, nil
}
