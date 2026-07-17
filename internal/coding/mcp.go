package coding

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// This file wires Z.AI's official Vision MCP Server (@z_ai/mcp-server) into
// each supported coding tool — the "manage MCP services" step of the
// official @z_ai/coding-helper wizard, which this package otherwise ports in
// full but previously had no equivalent for. The server exposes
// GLM-4.6V-backed vision tools (screenshot OCR, error-screenshot diagnosis,
// diagram/chart understanding, general image/video analysis) to MCP-aware
// clients; see https://docs.z.ai/devpack/mcp/vision-mcp-server.
//
// Deliberately scoped to just this one official server, not a general
// "register any MCP server" manager — every tool below gets the exact same
// npx-launched entry, only the file, top-level key, and entry shape differ.

const (
	// ZAIMCPPackage is the npm package the Vision MCP Server ships as,
	// launched on demand via `npx -y`— no separate install step needed
	// beyond having Node.js (docs.z.ai currently recommends >=22.0.0; the
	// npm package page itself only requires 18+) and npx on PATH.
	ZAIMCPPackage = "@z_ai/mcp-server"

	// zaiMCPServerName is the key/name every tool's config registers the
	// server under.
	zaiMCPServerName = "zai-mcp-server"

	// zaiMCPModeEnv is required alongside the API key so the server talks
	// to the international z.ai platform rather than open.bigmodel.cn.
	zaiMCPModeEnv = "ZAI"
)

// HasNPX reports whether npx (and therefore a usable Node.js toolchain) is on
// PATH. Informational only, like Tool.IsInstalled — coding mcp add still
// writes the config even if this is false, since the config is valid the
// moment npx becomes available; callers should surface this as a warning,
// not refuse to proceed.
func HasNPX() bool {
	_, err := exec.LookPath("npx")
	return err == nil
}

// zaiMCPEnv is the env block every tool's entry sets for the server process.
func zaiMCPEnv(apiKey string) map[string]any {
	return map[string]any{
		"Z_AI_API_KEY": apiKey,
		"Z_AI_MODE":    zaiMCPModeEnv,
	}
}

// stdioMCPEntry is the entry shape Claude Code and Crush both use verbatim.
func stdioMCPEntry(apiKey string) map[string]any {
	return map[string]any{
		"type":    "stdio",
		"command": "npx",
		"args":    []any{"-y", ZAIMCPPackage},
		"env":     zaiMCPEnv(apiKey),
	}
}

// bareMCPEntry is the entry shape Factory Droid and Cursor both use verbatim
// (command/args/env, no "type" field — neither tool's real config includes one).
func bareMCPEntry(apiKey string) map[string]any {
	return map[string]any{
		"command": "npx",
		"args":    []any{"-y", ZAIMCPPackage},
		"env":     zaiMCPEnv(apiKey),
	}
}

// storeMCPEntry writes entry into path's servers object under key
// (e.g. "mcpServers" or "mcp"), preserving every other entry already there.
func storeMCPEntry(path, key string, entry map[string]any) error {
	m, err := readJSONMap(path)
	if err != nil {
		return err
	}
	servers := objectField(m, key)
	servers[zaiMCPServerName] = entry
	m[key] = servers
	return writeJSONMap(path, m)
}

// removeMCPEntry deletes the Vision MCP Server entry from path's servers
// object under key, if present, preserving everything else in the file.
func removeMCPEntry(path, key string) error {
	m, err := readJSONMap(path)
	if err != nil {
		return err
	}
	if servers, ok := m[key].(map[string]any); ok {
		delete(servers, zaiMCPServerName)
		if len(servers) == 0 {
			delete(m, key)
		} else {
			m[key] = servers
		}
	}
	return writeJSONMap(path, m)
}

// detectMCPEntry reports whether the Vision MCP Server entry exists in
// path's servers object under key.
func detectMCPEntry(path, key string) (bool, error) {
	m, err := readJSONMap(path)
	if err != nil {
		return false, err
	}
	servers, ok := m[key].(map[string]any)
	if !ok {
		return false, nil
	}
	_, configured := servers[zaiMCPServerName]
	return configured, nil
}

// --- Claude Code (~/.claude.json — the same file LoadClaudeCodeOpts uses
// for hasCompletedOnboarding, not ~/.claude/settings.json where the
// credential env vars live; Claude Code reads MCP servers from a different
// file than provider env vars) ---

func mcpConfigPathClaudeCode(home string) string {
	return filepath.Join(home, ".claude.json")
}

// LoadMCPClaudeCode registers the Vision MCP Server for Claude Code.
func LoadMCPClaudeCode(home, apiKey string) error {
	return storeMCPEntry(mcpConfigPathClaudeCode(home), "mcpServers", stdioMCPEntry(apiKey))
}

// UnloadMCPClaudeCode removes the Vision MCP Server entry, if present.
func UnloadMCPClaudeCode(home string) error {
	return removeMCPEntry(mcpConfigPathClaudeCode(home), "mcpServers")
}

// DetectMCPClaudeCode reports whether the Vision MCP Server is registered.
func DetectMCPClaudeCode(home string) (bool, error) {
	return detectMCPEntry(mcpConfigPathClaudeCode(home), "mcpServers")
}

// --- OpenCode (opencode.json, "mcp" key — distinct shape from every other
// tool: type "local", command as an array, env key is "environment") ---

// LoadMCPOpenCode registers the Vision MCP Server for OpenCode.
func LoadMCPOpenCode(home, apiKey string) error {
	entry := map[string]any{
		"type":        "local",
		"command":     []any{"npx", "-y", ZAIMCPPackage},
		"environment": zaiMCPEnv(apiKey),
	}
	return storeMCPEntry(Tools[1].ConfigPath(home), "mcp", entry)
}

// UnloadMCPOpenCode removes the Vision MCP Server entry, if present.
func UnloadMCPOpenCode(home string) error {
	return removeMCPEntry(Tools[1].ConfigPath(home), "mcp")
}

// DetectMCPOpenCode reports whether the Vision MCP Server is registered.
func DetectMCPOpenCode(home string) (bool, error) {
	return detectMCPEntry(Tools[1].ConfigPath(home), "mcp")
}

// --- Crush (crush.json, "mcp" key — same stdio/command+args/env shape as
// Claude Code, just under a different top-level key) ---

// LoadMCPCrush registers the Vision MCP Server for Crush.
func LoadMCPCrush(home, apiKey string) error {
	return storeMCPEntry(Tools[2].ConfigPath(home), "mcp", stdioMCPEntry(apiKey))
}

// UnloadMCPCrush removes the Vision MCP Server entry, if present.
func UnloadMCPCrush(home string) error {
	return removeMCPEntry(Tools[2].ConfigPath(home), "mcp")
}

// DetectMCPCrush reports whether the Vision MCP Server is registered.
func DetectMCPCrush(home string) (bool, error) {
	return detectMCPEntry(Tools[2].ConfigPath(home), "mcp")
}

// --- Factory Droid (~/.factory/mcp.json — a separate file from
// ~/.factory/settings.json, which is where the credential customModels
// entries live; "mcpServers" key, no "type" field in Factory's own examples) ---

func mcpConfigPathFactoryDroid(home string) string {
	return filepath.Join(home, ".factory", "mcp.json")
}

// LoadMCPFactoryDroid registers the Vision MCP Server for Factory Droid.
func LoadMCPFactoryDroid(home, apiKey string) error {
	return storeMCPEntry(mcpConfigPathFactoryDroid(home), "mcpServers", bareMCPEntry(apiKey))
}

// UnloadMCPFactoryDroid removes the Vision MCP Server entry, if present.
func UnloadMCPFactoryDroid(home string) error {
	return removeMCPEntry(mcpConfigPathFactoryDroid(home), "mcpServers")
}

// DetectMCPFactoryDroid reports whether the Vision MCP Server is registered.
func DetectMCPFactoryDroid(home string) (bool, error) {
	return detectMCPEntry(mcpConfigPathFactoryDroid(home), "mcpServers")
}

// --- Cursor (a sibling mcp.json next to settings.json, same directory
// cursorConfigDir resolves for both). Cursor's own docs describe its MCP
// format as Claude-Desktop-compatible, which is where the "env" field in
// bareMCPEntry comes from — Z.AI's own docs don't explicitly list Cursor as
// a supported Vision MCP client, so this is the generic MCP shape applied to
// Z.AI's server rather than a Z.AI-confirmed integration; worth a live check. ---

func mcpConfigPathCursor(home string) string {
	return filepath.Join(cursorConfigDir(home), "mcp.json")
}

// LoadMCPCursor registers the Vision MCP Server for Cursor.
func LoadMCPCursor(home, apiKey string) error {
	return storeMCPEntry(mcpConfigPathCursor(home), "mcpServers", bareMCPEntry(apiKey))
}

// UnloadMCPCursor removes the Vision MCP Server entry, if present.
func UnloadMCPCursor(home string) error {
	return removeMCPEntry(mcpConfigPathCursor(home), "mcpServers")
}

// DetectMCPCursor reports whether the Vision MCP Server is registered.
func DetectMCPCursor(home string) (bool, error) {
	return detectMCPEntry(mcpConfigPathCursor(home), "mcpServers")
}

// --- Dispatch by tool ---

// LoadMCP registers Z.AI's Vision MCP Server for the named tool.
func LoadMCP(home, toolID, apiKey string) error {
	t, err := FindTool(toolID)
	if err != nil {
		return err
	}
	switch t.ID {
	case "claude-code":
		return LoadMCPClaudeCode(home, apiKey)
	case "opencode":
		return LoadMCPOpenCode(home, apiKey)
	case "crush":
		return LoadMCPCrush(home, apiKey)
	case "factory-droid":
		return LoadMCPFactoryDroid(home, apiKey)
	case "cursor":
		return LoadMCPCursor(home, apiKey)
	}
	return fmt.Errorf("unsupported tool %q", toolID)
}

// UnloadMCP removes Z.AI's Vision MCP Server entry from the named tool.
func UnloadMCP(home, toolID string) error {
	t, err := FindTool(toolID)
	if err != nil {
		return err
	}
	switch t.ID {
	case "claude-code":
		return UnloadMCPClaudeCode(home)
	case "opencode":
		return UnloadMCPOpenCode(home)
	case "crush":
		return UnloadMCPCrush(home)
	case "factory-droid":
		return UnloadMCPFactoryDroid(home)
	case "cursor":
		return UnloadMCPCursor(home)
	}
	return fmt.Errorf("unsupported tool %q", toolID)
}

// DetectMCPConfigured reports whether the named tool has Z.AI's Vision MCP
// Server registered.
func DetectMCPConfigured(home, toolID string) (bool, error) {
	t, err := FindTool(toolID)
	if err != nil {
		return false, err
	}
	switch t.ID {
	case "claude-code":
		return DetectMCPClaudeCode(home)
	case "opencode":
		return DetectMCPOpenCode(home)
	case "crush":
		return DetectMCPCrush(home)
	case "factory-droid":
		return DetectMCPFactoryDroid(home)
	case "cursor":
		return DetectMCPCursor(home)
	}
	return false, fmt.Errorf("unsupported tool %q", toolID)
}
