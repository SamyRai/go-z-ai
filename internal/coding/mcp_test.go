package coding

import (
	"errors"
	"os"
	"slices"
	"testing"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// Every tool registers the official servers it supports, detects them, and
// removes them without touching other entries in the same file.
func TestToolsMCPLoadDetectUnload(t *testing.T) {
	for _, tool := range Tools {
		t.Run(tool.ID, func(t *testing.T) {
			home := t.TempDir()
			servers := tool.SupportedMCPServers()
			writeConfig(t, tool.mcp.path(home), map[string]any{tool.mcp.key: map[string]any{"mine": map[string]any{"command": "x"}}, "other": true})
			if err := tool.LoadMCP(home, PlanGlobal, "k", servers); err != nil {
				t.Fatalf("LoadMCP: %v", err)
			}
			found, err := tool.MCPConfigured(home)
			if err != nil || len(found) != len(servers) {
				t.Fatalf("MCPConfigured = %v, %v", found, err)
			}
			if err := tool.UnloadMCP(home, servers); err != nil {
				t.Fatalf("UnloadMCP: %v", err)
			}
			m := readConfig(t, tool.mcp.path(home))
			entries := m[tool.mcp.key].(map[string]any)
			if _, ok := entries["mine"]; !ok || len(entries) != 1 || m["other"] != true {
				t.Errorf("user's entries disturbed: %v", m)
			}
		})
	}
}

// The Vision server's mode follows the plan's region (ZHIPU for China), and
// hosted servers use the region's MCP URL with a bearer header.
func TestMCPEntriesFollowRegion(t *testing.T) {
	home := t.TempDir()
	if err := claudeCode.LoadMCP(home, PlanChina, "k", MCPServers); err != nil {
		t.Fatalf("LoadMCP: %v", err)
	}
	servers := readConfig(t, claudeStatePath(home))["mcpServers"].(map[string]any)
	vision := servers["zai-mcp-server"].(map[string]any)
	if vision["type"] != "stdio" || vision["env"].(map[string]any)["Z_AI_MODE"] != "ZHIPU" {
		t.Errorf("vision entry %v", vision)
	}
	search := servers["web-search-prime"].(map[string]any)
	if search["type"] != "http" || search["url"] != client.RegionChina.MCPServerURL("web_search_prime") {
		t.Errorf("web-search-prime entry %v", search)
	}
	if search["headers"].(map[string]any)["Authorization"] != "Bearer k" {
		t.Errorf("missing bearer header: %v", search)
	}
}

func TestFindMCPServers(t *testing.T) {
	all, err := FindMCPServers()
	if err != nil || len(all) != len(MCPServers) {
		t.Fatalf("FindMCPServers() = %v, %v", all, err)
	}
	some, err := FindMCPServers("zread", "zai-mcp-server")
	if err != nil || !slices.EqualFunc(some, []string{"zread", "zai-mcp-server"}, func(s MCPServer, id string) bool { return s.ID == id }) {
		t.Errorf("FindMCPServers(zread, zai-mcp-server) = %v, %v", some, err)
	}
	if _, err := FindMCPServers("nope"); err == nil {
		t.Error("expected an error for an unknown server")
	}
}

// Codex takes only the local Vision server; asking for a hosted one fails
// before anything is written.
func TestCodexRejectsHostedMCP(t *testing.T) {
	if got := codex.SupportedMCPServers(); len(got) != 1 || got[0].IsRemote() {
		t.Fatalf("SupportedMCPServers = %v, want only the local Vision server", got)
	}
	home := t.TempDir()
	if err := codex.LoadMCP(home, PlanGlobal, "k", MCPServers); err == nil {
		t.Fatal("expected an error for hosted servers")
	}
	if _, err := os.Stat(codexConfigPath(home)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("config written despite the error: %v", err)
	}
	for _, tool := range Tools {
		if tool.ID != CodexID && len(tool.SupportedMCPServers()) != len(MCPServers) {
			t.Errorf("%s should support every official server", tool.ID)
		}
	}
}
