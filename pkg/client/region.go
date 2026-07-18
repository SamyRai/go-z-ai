package client

// Region identifies which of Z.AI's two regional gateways a request targets.
// The same GLM model family is served from both; pick the one that matches
// where the API key was issued (Global keys -> api.z.ai, China keys ->
// open.bigmodel.cn) to avoid connectivity or authentication friction.
//
// The Region selects the host for services whose base URL is otherwise
// hardcoded to api.z.ai (monitor, biz, agents); it does NOT override an
// explicit Config.BaseURL (the chat/completions surface) or BigModelBaseURL
// (embeddings/moderations, which always use the China host regardless).
type Region string

const (
	// RegionGlobal is the international gateway (api.z.ai). This is the
	// default and the historical behavior; existing callers see no change.
	RegionGlobal Region = "global"
	// RegionChina is the China-mainland gateway (open.bigmodel.cn). Use this
	// when the API key was issued on open.bigmodel.cn so monitor/biz/agents
	// calls land on the matching host.
	RegionChina Region = "china"
)

// monitorBaseURL returns the monitor host for a region. The coding-plan
// quota/usage endpoints (/usage/quota/limit, /usage/model-usage,
// /usage/tool-usage) live here. NOT VERIFIED LIVE that the China monitor host
// mirrors the api.z.ai one — recorded as the expected path for a
// glm_coding_plan_china key; if a live capture disagrees, pin it here.
func (r Region) monitorBaseURL() string {
	if r == RegionChina {
		return ChinaMonitorBaseURL
	}
	return MonitorBaseURL
}

// bizBaseURL returns the biz host for a region (the /account/info and
// /account/status endpoints). Same caveat as monitorBaseURL for RegionChina.
func (r Region) bizBaseURL() string {
	if r == RegionChina {
		return ChinaBizBaseURL
	}
	return BizBaseURL
}

// agentsBaseURL returns the agents host for a region. The agents API uses a
// bare root (no /paas/v4); nesting under the chat base 404s (live-verified
// for the global host). The China mirror follows the same shape.
func (r Region) agentsBaseURL() string {
	if r == RegionChina {
		return ChinaAgentsBaseURL
	}
	return AgentsBaseURL
}
