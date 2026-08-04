package accounts

import (
	"context"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// ProbeType classifies an API key's account type with a single free
// (no token cost) call to the coding-plan-only monitor/quota endpoint,
// instead of firing a real billed chat completion.
//
// It probes BOTH regional gateways (Global api.z.ai and China
// open.bigmodel.cn): a coding-plan key issued on the China host returns a
// coding-plan success only there, so probing only the global host (as a prior
// version did) systematically misclassified China keys as pay_as_you_go.
//
// A successful, well-formed response from EITHER gateway classifies the key as
// coding_plan with confirmed=true. Anything else (non-200, unsuccessful
// response, or a decode/network failure on both) falls back to pay_as_you_go
// with confirmed=false — this is an inference by elimination, not a positive
// confirmation, since no endpoint exists that is known to work for
// pay-as-you-go keys specifically.
func ProbeType(ctx context.Context, apiKey string) (accountType client.AccountType, confirmed bool, err error) {
	for _, region := range []client.Region{client.RegionGlobal, client.RegionChina} {
		c, err := client.NewClient(client.Config{APIKey: apiKey, Region: region})
		if err != nil {
			return "", false, err
		}
		if at, ok := probeType(ctx, c); ok {
			return at, true, nil
		}
	}
	return client.AccountTypePayAsYouGo, false, nil
}

// probeType is the single-region classification step against an already-built
// client. It returns (coding_plan, true) only on a clean success; otherwise
// (pay_as_you_go, false) so the caller can try the next region. It's split out
// so tests can inject an HTTP transport: GetQuotaLimit targets a hardcoded
// monitor base URL that Config.BaseURL can't redirect, so a canned transport
// on Config.HTTPClient is the only seam.
func probeType(ctx context.Context, c *client.Client) (client.AccountType, bool) {
	quota, callErr := c.Quota().GetQuotaLimit(ctx)
	if callErr == nil && quota != nil && quota.Success && quota.Data.Level != "" {
		return client.AccountTypeCodingPlan, true
	}
	return client.AccountTypePayAsYouGo, false
}
