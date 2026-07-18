package client

import "testing"

// RegionGlobal must resolve to the api.z.ai hosts; RegionChina must resolve to
// the open.bigmodel.cn mirrors. This is the core parity contract.
func TestRegionBaseURLResolution(t *testing.T) {
	cases := []struct {
		name         string
		region       Region
		monitor, biz string
		agents       string
	}{
		{
			name:    "global",
			region:  RegionGlobal,
			monitor: MonitorBaseURL,
			biz:     BizBaseURL,
			agents:  AgentsBaseURL,
		},
		{
			name:    "china",
			region:  RegionChina,
			monitor: ChinaMonitorBaseURL,
			biz:     ChinaBizBaseURL,
			agents:  ChinaAgentsBaseURL,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.region.monitorBaseURL(); got != tc.monitor {
				t.Errorf("monitor: got %q, want %q", got, tc.monitor)
			}
			if got := tc.region.bizBaseURL(); got != tc.biz {
				t.Errorf("biz: got %q, want %q", got, tc.biz)
			}
			if got := tc.region.agentsBaseURL(); got != tc.agents {
				t.Errorf("agents: got %q, want %q", got, tc.agents)
			}
		})
	}
}

// An unknown Region value must fall back to the global hosts — never panic,
// never return an empty string. This makes a config typo non-fatal.
func TestRegionUnknownFallsBackToGlobal(t *testing.T) {
	unknown := Region("mars")
	if got := unknown.monitorBaseURL(); got != MonitorBaseURL {
		t.Errorf("monitor for unknown region: got %q, want %q", got, MonitorBaseURL)
	}
	if got := unknown.bizBaseURL(); got != BizBaseURL {
		t.Errorf("biz for unknown region: got %q, want %q", got, BizBaseURL)
	}
	if got := unknown.agentsBaseURL(); got != AgentsBaseURL {
		t.Errorf("agents for unknown region: got %q, want %q", got, AgentsBaseURL)
	}
}

// An empty Region (the zero value a Config starts with) must resolve to global
// — this is the back-compat guarantee for callers who never set Region.
func TestRegionEmptyDefaultsGlobal(t *testing.T) {
	var r Region
	if got := r.monitorBaseURL(); got != MonitorBaseURL {
		t.Errorf("empty Region monitor: got %q, want %q", got, MonitorBaseURL)
	}
}

// China hosts must be on open.bigmodel.cn (not api.z.ai), and global hosts on
// api.z.ai. A drift in the constants would break a region selection silently.
func TestRegionHostsMatchExpectedPlatform(t *testing.T) {
	for h, want := range map[string]string{
		ChinaMonitorBaseURL: "open.bigmodel.cn",
		ChinaBizBaseURL:     "open.bigmodel.cn",
		ChinaAgentsBaseURL:  "open.bigmodel.cn",
		MonitorBaseURL:      "api.z.ai",
		BizBaseURL:          "api.z.ai",
		AgentsBaseURL:       "api.z.ai",
	} {
		if !contains(h, want) {
			t.Errorf("host %q does not contain %q", h, want)
		}
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
