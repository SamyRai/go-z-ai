// Package coding manages GLM Coding Plan credentials and writes them into
// supported coding tools (Claude Code, OpenCode, Crush, Factory Droid) in the
// same format as Z.AI's official @z_ai/coding-helper, plus Z.AI's official
// MCP servers. The credential store at ~/.chelper/config.yaml is
// byte-compatible with the official helper, so the two can share state.
//
// Every endpoint comes from client.Region and every model default from the
// client's model catalog; this package owns only the tool-specific file
// formats.
package coding

import "github.com/SamyRai/go-z-ai/pkg/client"

// Plan identifiers mirror @z_ai/coding-helper's plan strings exactly.
const (
	// PlanGlobal is the international GLM Coding Plan (api.z.ai).
	PlanGlobal = "glm_coding_plan_global"
	// PlanChina is the China GLM Coding Plan (open.bigmodel.cn).
	PlanChina = "glm_coding_plan_china"
)

// Plans lists the valid plan identifiers.
var Plans = []string{PlanGlobal, PlanChina}

// IsValidPlan reports whether p is a recognized plan identifier.
func IsValidPlan(p string) bool {
	return p == PlanGlobal || p == PlanChina
}

// Region returns the gateway a plan's keys belong to.
func Region(plan string) client.Region {
	if plan == PlanChina {
		return client.RegionChina
	}
	return client.RegionGlobal
}

// DisplayName returns a human-readable plan name.
func DisplayName(p string) string {
	if label := RegionLabel(p); label != "" {
		return "GLM Coding Plan (" + label + ")"
	}
	return "Unknown plan"
}

// RegionLabel returns the short region label for a plan ("Global" / "China"),
// or "" for an unrecognized plan.
func RegionLabel(p string) string {
	switch p {
	case PlanGlobal:
		return "Global"
	case PlanChina:
		return "China"
	}
	return ""
}

// planFromBaseURL infers a plan from a configured endpoint URL, returning
// ("", false) when it is not one of the plan endpoints.
func planFromBaseURL(baseURL string) (string, bool) {
	for _, plan := range Plans {
		r := Region(plan)
		switch baseURL {
		case r.AnthropicBaseURL(), r.CodingBaseURL(), r.ResponsesBaseURL():
			return plan, true
		}
	}
	return "", false
}
