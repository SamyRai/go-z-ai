package client

import "context"

// AccountService reads account data from the biz API (Region.BizBaseURL).
//
// NOT VERIFIED LIVE in this repository: Z.AI does not document the biz API.
// The two endpoints below are the ones Z.AI's own ZCode client and many
// community usage tools call (checked 2026-10-02); their field names come
// from those clients. Pin them with a cassette (ZAI_RECORD=1) when you can.
type AccountService struct {
	client *Client
}

// Balance is the pay-as-you-go wallet. Amounts are in the region's billing
// currency (USD on api.z.ai, CNY on open.bigmodel.cn); the API sends no
// currency field.
type Balance struct {
	Balance          float64 `json:"balance"`
	AvailableBalance float64 `json:"availableBalance"`
	RechargeAmount   float64 `json:"rechargeAmount"`
	GiveAmount       float64 `json:"giveAmount"` // promotional credit
	TotalSpendAmount float64 `json:"totalSpendAmount"`
	FrozenBalance    float64 `json:"frozenBalance"`
}

// Subscription is one GLM Coding Plan subscription.
type Subscription struct {
	ID               string `json:"id"`
	ProductName      string `json:"productName"` // e.g. "GLM Coding Pro"
	Status           string `json:"status"`      // e.g. "VALID"
	BillingCycle     string `json:"billingCycle"`
	Valid            string `json:"valid"`         // "start-end" validity range
	NextRenewTime    string `json:"nextRenewTime"` // YYYY-MM-DD
	AutoRenew        int    `json:"autoRenew"`     // 1 = on
	InCurrentPeriod  bool   `json:"inCurrentPeriod"`
	CurrentRenewTime string `json:"currentRenewTime"`
}

// Balance returns the pay-as-you-go wallet
// (GET /account/query-customer-account-report).
func (s *AccountService) Balance(ctx context.Context) (*Balance, error) {
	env, err := fetchEnvelope[Balance](ctx, s.client, s.request("/account/query-customer-account-report"), "account balance")
	if err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// Subscriptions returns the account's Coding Plan subscriptions
// (GET /subscription/list); empty when it has none.
func (s *AccountService) Subscriptions(ctx context.Context) ([]Subscription, error) {
	env, err := fetchEnvelope[[]Subscription](ctx, s.client, s.request("/subscription/list"), "subscriptions")
	if err != nil {
		return nil, err
	}
	return env.Data, nil
}

// request builds a GET against the region's biz root.
func (s *AccountService) request(path string) apiRequest {
	return apiRequest{method: "GET", baseURL: s.client.config.Region.BizBaseURL(), path: path, service: "account"}
}
