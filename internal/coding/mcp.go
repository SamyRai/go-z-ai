package coding

import (
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// MCPServer is one of Z.AI's official MCP servers for the GLM Coding Plan
// (docs.z.ai/devpack/mcp): the local Vision server, launched with npx, and
// three hosted streamable-HTTP servers. Calls to any of them draw plan quota.
type MCPServer struct {
	ID          string // the name tools register it under
	Description string
	// remote is the hosted server's name in client.Region.MCPServerURL; empty
	// for the local Vision server.
	remote string
}

// VisionMCPPackage is the npm package of the local Vision MCP server.
const VisionMCPPackage = "@z_ai/mcp-server"

// MCPServers is the registry of official servers.
var MCPServers = []MCPServer{
	{ID: "zai-mcp-server", Description: "Vision: screenshots, diagrams, UI, images and video (local, npx)"},
	{ID: "web-search-prime", Description: "Web search", remote: "web_search_prime"},
	{ID: "web-reader", Description: "Fetch and read web pages", remote: "web_reader"},
	{ID: "zread", Description: "Read and search GitHub repositories", remote: "zread"},
}

// FindMCPServers resolves server IDs; no IDs means every official server.
func FindMCPServers(ids ...string) ([]MCPServer, error) {
	if len(ids) == 0 {
		return MCPServers, nil
	}
	out := make([]MCPServer, 0, len(ids))
	for _, id := range ids {
		i := slices.IndexFunc(MCPServers, func(s MCPServer) bool { return s.ID == id })
		if i < 0 {
			return nil, fmt.Errorf("unknown MCP server %q (known: %s)", id, strings.Join(mcpServerIDs(), ", "))
		}
		out = append(out, MCPServers[i])
	}
	return out, nil
}

// mcpServerIDs lists the official server IDs.
func mcpServerIDs() []string {
	ids := make([]string, len(MCPServers))
	for i, s := range MCPServers {
		ids[i] = s.ID
	}
	return ids
}

// IsRemote reports whether the server is hosted (streamable HTTP).
func (s MCPServer) IsRemote() bool { return s.remote != "" }

// HasNPX reports whether npx is on PATH — needed only by the local Vision
// server. Informational: configs are written regardless.
func HasNPX() bool {
	_, err := exec.LookPath("npx")
	return err == nil
}

// mcpTarget describes where and in which shape a tool registers MCP servers.
type mcpTarget struct {
	path   func(home string) string
	key    string // top-level object holding the servers
	local  func(command string, args []string, env map[string]any) map[string]any
	remote func(url string, headers map[string]any) map[string]any
}

// entry builds s's config entry for plan in this tool's shape.
func (t mcpTarget) entry(s MCPServer, plan, apiKey string) map[string]any {
	region := Region(plan)
	if s.IsRemote() {
		return t.remote(region.MCPServerURL(s.remote), map[string]any{"Authorization": "Bearer " + apiKey})
	}
	mode := "ZAI" // the Vision server's switch between api.z.ai and open.bigmodel.cn
	if region == client.RegionChina {
		mode = "ZHIPU"
	}
	return t.local("npx", []string{"-y", VisionMCPPackage}, map[string]any{"Z_AI_API_KEY": apiKey, "Z_AI_MODE": mode})
}

// LoadMCP registers servers in the tool's config for plan.
func (t Tool) LoadMCP(home, plan, apiKey string, servers []MCPServer) error {
	return editJSONMap(t.mcp.path(home), func(m map[string]any) {
		entries := objectField(m, t.mcp.key)
		for _, s := range servers {
			entries[s.ID] = t.mcp.entry(s, plan, apiKey)
		}
	})
}

// UnloadMCP removes servers from the tool's config, leaving others alone.
func (t Tool) UnloadMCP(home string, servers []MCPServer) error {
	return editJSONMap(t.mcp.path(home), func(m map[string]any) {
		for _, s := range servers {
			deleteFromObject(m, t.mcp.key, s.ID)
		}
	})
}

// MCPConfigured returns the official servers registered in the tool's config.
func (t Tool) MCPConfigured(home string) ([]MCPServer, error) {
	m, err := readJSONMap(t.mcp.path(home))
	if err != nil {
		return nil, err
	}
	entries, _ := m[t.mcp.key].(map[string]any)
	var found []MCPServer
	for _, s := range MCPServers {
		if _, ok := entries[s.ID]; ok {
			found = append(found, s)
		}
	}
	return found, nil
}

// stdioEntry and httpEntry are the {type: stdio|http} shapes Claude Code,
// Crush, and Factory Droid share.
func stdioEntry(command string, args []string, env map[string]any) map[string]any {
	return map[string]any{"type": "stdio", "command": command, "args": anySlice(args), "env": env}
}

func httpEntry(url string, headers map[string]any) map[string]any {
	return map[string]any{"type": "http", "url": url, "headers": headers}
}

func anySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}
