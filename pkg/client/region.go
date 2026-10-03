package client

import "strings"

// Region identifies which of Z.AI's two regional gateways a request targets.
// The same GLM model family is served from both; pick the one that matches
// where the API key was issued (Global keys -> api.z.ai, China keys ->
// open.bigmodel.cn) to avoid connectivity or authentication friction.
//
// Region is the single owner of the gateway URL layout: every base URL in
// this package (and every config writer in internal/coding) is derived from
// Region.Host plus one of the API-root paths below, so a host or path change
// is a one-line edit.
type Region string

const (
	// RegionGlobal is the international gateway (api.z.ai). This is the
	// default; the zero Region and any unrecognized value resolve to it.
	RegionGlobal Region = "global"
	// RegionChina is the China-mainland gateway (open.bigmodel.cn).
	RegionChina Region = "china"
)

// Gateway hosts.
const (
	GlobalHost = "https://api.z.ai"
	ChinaHost  = "https://open.bigmodel.cn"
)

// API roots under a gateway host. The agents API uses the bare /api root —
// nesting /v1/agents under the chat-completions base 404s (live-verified on
// the global host). The OpenAI Responses protocol (what Codex speaks) lives
// under /api/v1 (docs.z.ai/devpack/tool/codex).
const (
	paasPath      = "/api/paas/v4"
	codingPath    = "/api/coding/paas/v4"
	anthropicPath = "/api/anthropic"
	monitorPath   = "/api/monitor"
	bizPath       = "/api/biz"
	agentsPath    = "/api"
	mcpPath       = "/api/mcp"
	responsesPath = "/api/v1"
)

// Global-gateway base URLs.
const (
	DefaultBaseURL   = GlobalHost + paasPath
	CodingBaseURL    = GlobalHost + codingPath
	AnthropicBaseURL = GlobalHost + anthropicPath
	MonitorBaseURL   = GlobalHost + monitorPath
	BizBaseURL       = GlobalHost + bizPath
	AgentsBaseURL    = GlobalHost + agentsPath
	ResponsesBaseURL = GlobalHost + responsesPath
)

// China-gateway base URLs. The China mirror serves the same OpenAPI surface
// as api.z.ai — live-verified 2026-07-11 for /models and /chat/completions
// (a single z.ai key authenticates identically on both platforms). The
// monitor/biz/agents mirrors follow the same path layout but are NOT
// VERIFIED LIVE; pin them with a cassette if you hold an entitled China key.
const (
	BigModelBaseURL       = ChinaHost + paasPath
	ChinaCodingBaseURL    = ChinaHost + codingPath
	ChinaAnthropicBaseURL = ChinaHost + anthropicPath
	ChinaMonitorBaseURL   = ChinaHost + monitorPath
	ChinaBizBaseURL       = ChinaHost + bizPath
	ChinaAgentsBaseURL    = ChinaHost + agentsPath
	ChinaResponsesBaseURL = ChinaHost + responsesPath
)

// ParseRegion maps a user-supplied region name to a Region. It accepts
// "china" (aliases "cn", "bigmodel") and "global" (aliases "west", ""),
// case-insensitively. An unknown value resolves to RegionGlobal rather than
// erroring, so a typo never blocks a command that doesn't touch a
// region-scoped service.
func ParseRegion(s string) Region {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "china", "cn", "bigmodel":
		return RegionChina
	default:
		return RegionGlobal
	}
}

// Host returns the gateway host for r. Unknown or empty values resolve to the
// global host.
func (r Region) Host() string {
	if r == RegionChina {
		return ChinaHost
	}
	return GlobalHost
}

// PaaSBaseURL is the pay-as-you-go OpenAI-compatible API root.
func (r Region) PaaSBaseURL() string { return r.Host() + paasPath }

// CodingBaseURL is the GLM Coding Plan OpenAI-compatible API root.
func (r Region) CodingBaseURL() string { return r.Host() + codingPath }

// AnthropicBaseURL is the Anthropic-compatible (Messages API) root.
func (r Region) AnthropicBaseURL() string { return r.Host() + anthropicPath }

// MonitorBaseURL is the root of the coding-plan quota/usage endpoints.
func (r Region) MonitorBaseURL() string { return r.Host() + monitorPath }

// BizBaseURL is the root of the account (biz) endpoints.
func (r Region) BizBaseURL() string { return r.Host() + bizPath }

// AgentsBaseURL is the root of the agents API (bare /api, see agentsPath).
func (r Region) AgentsBaseURL() string { return r.Host() + agentsPath }

// ResponsesBaseURL is the OpenAI Responses-protocol root (POST /responses),
// the endpoint Codex is configured against.
func (r Region) ResponsesBaseURL() string { return r.Host() + responsesPath }

// ConsoleURL is the web console where the region's keys, billing, and
// subscriptions are managed.
func (r Region) ConsoleURL() string {
	if r == RegionChina {
		return "https://bigmodel.cn"
	}
	return "https://z.ai"
}

// MCPServerURL is the streamable-HTTP endpoint of one of Z.AI's hosted MCP
// servers (e.g. "web_search_prime", "web_reader", "zread"), authenticated
// with "Authorization: Bearer <key>".
func (r Region) MCPServerURL(name string) string { return r.Host() + mcpPath + "/" + name + "/mcp" }

// BaseURLFor returns the chat API root an account type uses in this region:
// the coding endpoint for a coding-plan key, the pay-as-you-go endpoint
// otherwise.
func (r Region) BaseURLFor(t AccountType) string {
	if t == AccountTypeCodingPlan {
		return r.CodingBaseURL()
	}
	return r.PaaSBaseURL()
}
