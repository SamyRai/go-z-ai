// Package coding is a Go port of Z.AI's official @z_ai/coding-helper ("chelper")
// CLI. It manages GLM Coding Plan credentials and loads them into supported
// coding tools (Claude Code, OpenCode, Crush, Factory Droid), writing each
// tool's native config in the exact format the official Node helper uses.
//
// The credential store at ~/.chelper/config.yaml is byte-compatible with the
// official helper, so the two tools can share state.
package coding

// Plan identifiers mirror @z_ai/coding-helper's plan strings exactly.
const (
	// PlanGlobal is the international GLM Coding Plan (api.z.ai).
	PlanGlobal = "glm_coding_plan_global"
	// PlanChina is the China GLM Coding Plan (open.bigmodel.cn).
	PlanChina = "glm_coding_plan_china"
)

// IsValidPlan reports whether p is a recognized plan identifier.
func IsValidPlan(p string) bool {
	return p == PlanGlobal || p == PlanChina
}

// DisplayName returns a human-readable plan name.
func DisplayName(p string) string {
	switch p {
	case PlanGlobal:
		return "GLM Coding Plan (Global)"
	case PlanChina:
		return "GLM Coding Plan (China)"
	default:
		return "Unknown plan"
	}
}

// CodingBaseURL is the OpenAI-compatible coding-plan endpoint a plan uses
// (https://api.z.ai/api/coding/paas/v4 or the open.bigmodel.cn mirror).
func CodingBaseURL(plan string) string {
	if plan == PlanChina {
		return "https://open.bigmodel.cn/api/coding/paas/v4"
	}
	return "https://api.z.ai/api/coding/paas/v4"
}

// AnthropicBaseURL is the Anthropic-compatible endpoint a plan uses
// (https://api.z.ai/api/anthropic or the open.bigmodel.cn mirror). Coding apps
// that speak the Anthropic protocol point here.
func AnthropicBaseURL(plan string) string {
	if plan == PlanChina {
		return "https://open.bigmodel.cn/api/anthropic"
	}
	return "https://api.z.ai/api/anthropic"
}

// planFromBaseURL infers a plan from a configured endpoint URL, returning
// ("", false) when it is not a known Z.AI/BigModel endpoint.
func planFromBaseURL(baseURL string) (string, bool) {
	switch baseURL {
	case "https://api.z.ai/api/anthropic", "https://api.z.ai/api/coding/paas/v4":
		return PlanGlobal, true
	case "https://open.bigmodel.cn/api/anthropic", "https://open.bigmodel.cn/api/coding/paas/v4":
		return PlanChina, true
	default:
		return "", false
	}
}
