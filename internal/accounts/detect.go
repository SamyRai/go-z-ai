package accounts

import (
	"context"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// ProbeType classifies an API key's account type with a single free
// (no token cost) call to the coding-plan-only monitor/quota endpoint,
// instead of firing a real billed chat completion.
//
// A successful, well-formed response classifies the key as coding_plan with
// confirmed=true. Anything else (non-200, unsuccessful response, or a
// decode/network failure) falls back to pay_as_you_go with confirmed=false —
// this is an inference by elimination, not a positive confirmation, since no
// endpoint exists that is known to work for pay-as-you-go keys specifically.
func ProbeType(ctx context.Context, apiKey string) (accountType client.AccountType, confirmed bool, err error) {
	c, err := client.NewClient(client.Config{APIKey: apiKey})
	if err != nil {
		return "", false, err
	}
	at, confirmed := probeType(ctx, c)
	return at, confirmed, nil
}

// probeType is ProbeType's classification step against an already-built client.
// It's split out so tests can inject an HTTP transport: GetQuotaLimit targets a
// hardcoded monitor base URL that Config.BaseURL can't redirect, so a canned
// transport on Config.HTTPClient is the only seam.
func probeType(ctx context.Context, c *client.Client) (client.AccountType, bool) {
	quota, callErr := c.Quota().GetQuotaLimit(ctx)
	if callErr == nil && quota != nil && quota.Success && quota.Data.Level != "" {
		return client.AccountTypeCodingPlan, true
	}
	return client.AccountTypePayAsYouGo, false
}
