package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// AccountType classifies a Z.AI API key.
type AccountType string

const (
	AccountTypePayAsYouGo AccountType = "pay_as_you_go"
	AccountTypeCodingPlan AccountType = "coding_plan"
)

// DetectedAccount is the result of classifying an API key.
type DetectedAccount struct {
	Type   AccountType `json:"type"`
	Region Region      `json:"region"`
	// BaseURL is the chat API root this key should use (Region.BaseURLFor).
	BaseURL string `json:"base_url"`
	// Confirmed is true when the coding-plan quota endpoint positively
	// identified a subscription. No endpoint positively identifies a
	// pay-as-you-go key, so that result is an inference by elimination and
	// is never confirmed.
	Confirmed bool `json:"confirmed"`
	// Level is the coding-plan tier the quota endpoint reports (e.g. "lite",
	// "pro", "max"); empty for pay-as-you-go keys.
	Level string `json:"level,omitempty"`
}

// DetectionService classifies the client's API key — account type and the
// regional gateway it belongs to. It is the single owner of that logic; the
// CLI and TUI both call it.
type DetectionService struct {
	client *Client
	mu     sync.Mutex
	cache  *DetectedAccount
}

// DetectAccountType classifies the client's key with a free call to the
// coding-plan-only quota endpoint — no tokens are spent and no quota is
// consumed. It probes the client's configured region first, then the other
// gateway, since a China-issued coding-plan key only answers on
// open.bigmodel.cn. A successful, well-formed quota response from either
// gateway confirms a coding plan; an API error from both means pay-as-you-go
// (unconfirmed). If neither gateway could be reached at all, the transport
// error is returned rather than guessing. The result is cached per client.
func (s *DetectionService) DetectAccountType(ctx context.Context) (*DetectedAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cache != nil {
		return s.cache, nil
	}

	home := ParseRegion(string(s.client.config.Region))
	answered := false
	var lastErr error
	for _, region := range []Region{home, otherRegion(home)} {
		level, err := s.probe(ctx, region)
		if err == nil {
			s.cache = &DetectedAccount{
				Type:      AccountTypeCodingPlan,
				Region:    region,
				BaseURL:   region.BaseURLFor(AccountTypeCodingPlan),
				Confirmed: true,
				Level:     level,
			}
			return s.cache, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		lastErr = err
		if _, isAPIErr := errors.AsType[*APIError](err); isAPIErr || errors.Is(err, errNotCodingPlan) {
			answered = true
		}
	}
	if !answered {
		return nil, fmt.Errorf("account detection: no gateway reachable: %w", lastErr)
	}
	s.cache = &DetectedAccount{
		Type:    AccountTypePayAsYouGo,
		Region:  home,
		BaseURL: home.BaseURLFor(AccountTypePayAsYouGo),
	}
	return s.cache, nil
}

// errNotCodingPlan reports a quota response that answered but did not
// identify a subscription.
var errNotCodingPlan = errors.New("quota endpoint did not identify a coding plan")

// probe asks region's quota endpoint about the key and returns the plan tier
// when it identifies a coding-plan subscription.
func (s *DetectionService) probe(ctx context.Context, region Region) (string, error) {
	var res QuotaLimitResponse
	r := apiRequest{method: "GET", baseURL: region.MonitorBaseURL(), path: QuotaLimitEndpoint, service: "detection"}
	if err := s.client.do(ctx, r, &res); err != nil {
		if _, decodeErr := errors.AsType[*decodeError](err); decodeErr {
			return "", errNotCodingPlan
		}
		return "", err
	}
	if !res.OK() || res.Data.Level == "" {
		return "", errNotCodingPlan
	}
	return res.Data.Level, nil
}

// otherRegion returns the gateway that is not r.
func otherRegion(r Region) Region {
	if r == RegionChina {
		return RegionGlobal
	}
	return RegionChina
}
