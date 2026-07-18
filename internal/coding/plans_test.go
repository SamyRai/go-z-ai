package coding

import "testing"

// CodingBaseURL / AnthropicBaseURL / MonitorBaseURL / BizBaseURL / AgentsBaseURL
// must return the open.bigmodel.cn mirror for PlanChina and the api.z.ai host
// for PlanGlobal. A drift here would route a China-plan key at the wrong host.
func TestPlanBaseURLsByRegion(t *testing.T) {
	cases := []struct {
		name     string
		plan     string
		platform string // expected substring
	}{
		{"coding/global", CodingBaseURL(PlanGlobal), "api.z.ai"},
		{"coding/china", CodingBaseURL(PlanChina), "open.bigmodel.cn"},
		{"anthropic/global", AnthropicBaseURL(PlanGlobal), "api.z.ai"},
		{"anthropic/china", AnthropicBaseURL(PlanChina), "open.bigmodel.cn"},
		{"monitor/global", MonitorBaseURL(PlanGlobal), "api.z.ai"},
		{"monitor/china", MonitorBaseURL(PlanChina), "open.bigmodel.cn"},
		{"biz/global", BizBaseURL(PlanGlobal), "api.z.ai"},
		{"biz/china", BizBaseURL(PlanChina), "open.bigmodel.cn"},
		{"agents/global", AgentsBaseURL(PlanGlobal), "api.z.ai"},
		{"agents/china", AgentsBaseURL(PlanChina), "open.bigmodel.cn"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !contains(c.plan, c.platform) {
				t.Errorf("%s: expected %q to contain %q", c.name, c.plan, c.platform)
			}
		})
	}
}

// AgentsBaseURL must be the bare root (no /paas/v4) — nesting the agents path
// under the chat base 404s (live-verified for the global host).
func TestAgentsBaseURLHasNoPaasV4(t *testing.T) {
	if contains(AgentsBaseURL(PlanGlobal), "/paas/v4") {
		t.Errorf("agents base must be bare root, got %q", AgentsBaseURL(PlanGlobal))
	}
	if contains(AgentsBaseURL(PlanChina), "/paas/v4") {
		t.Errorf("agents base must be bare root, got %q", AgentsBaseURL(PlanChina))
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
