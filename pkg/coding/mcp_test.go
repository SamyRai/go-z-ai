package coding

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClaudeCodeMCPLoadDetectUnload(t *testing.T) {
	home := t.TempDir()

	if err := LoadMCP(home, "claude-code", "vision-key"); err != nil {
		t.Fatalf("LoadMCP: %v", err)
	}

	m := readJSON(t, mcpConfigPathClaudeCode(home))
	servers := m["mcpServers"].(map[string]interface{})
	entry := servers[zaiMCPServerName].(map[string]interface{})
	if entry["type"] != "stdio" {
		t.Errorf("type = %v, want stdio", entry["type"])
	}
	if entry["command"] != "npx" {
		t.Errorf("command = %v, want npx", entry["command"])
	}
	env := entry["env"].(map[string]interface{})
	if env["Z_AI_API_KEY"] != "vision-key" || env["Z_AI_MODE"] != "ZAI" {
		t.Errorf("unexpected env: %+v", env)
	}

	configured, err := DetectMCPConfigured(home, "claude-code")
	if err != nil || !configured {
		t.Fatalf("DetectMCPConfigured = %v, %v; want true, nil", configured, err)
	}

	if err := UnloadMCP(home, "claude-code"); err != nil {
		t.Fatalf("UnloadMCP: %v", err)
	}
	configured, _ = DetectMCPConfigured(home, "claude-code")
	if configured {
		t.Fatal("should not be configured after unload")
	}
}

// Claude Code's MCP config lives in the same ~/.claude.json file
// LoadClaudeCodeOpts uses for hasCompletedOnboarding — confirm LoadMCP
// doesn't clobber it, and vice versa.
func TestClaudeCodeMCPPreservesOnboardingFlag(t *testing.T) {
	home := t.TempDir()
	path := mcpConfigPathClaudeCode(home)
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	_ = os.WriteFile(path, []byte(`{"hasCompletedOnboarding":true,"other":1}`), 0o600)

	if err := LoadMCP(home, "claude-code", "k"); err != nil {
		t.Fatalf("LoadMCP: %v", err)
	}
	m := readJSON(t, path)
	if m["hasCompletedOnboarding"] != true {
		t.Errorf("hasCompletedOnboarding lost: %v", m["hasCompletedOnboarding"])
	}
	if m["other"] != float64(1) {
		t.Errorf("unrelated key lost: %v", m["other"])
	}

	if err := UnloadMCP(home, "claude-code"); err != nil {
		t.Fatalf("UnloadMCP: %v", err)
	}
	m = readJSON(t, path)
	if m["hasCompletedOnboarding"] != true || m["other"] != float64(1) {
		t.Errorf("unrelated keys not preserved after unload: %+v", m)
	}
}

func TestOpenCodeMCPLoadDetectUnload(t *testing.T) {
	home := t.TempDir()

	if err := LoadMCP(home, "opencode", "vision-key"); err != nil {
		t.Fatalf("LoadMCP: %v", err)
	}

	m := readJSON(t, Tools[1].ConfigPath(home))
	servers := m["mcp"].(map[string]interface{})
	entry := servers[zaiMCPServerName].(map[string]interface{})
	if entry["type"] != "local" {
		t.Errorf("type = %v, want local", entry["type"])
	}
	cmdArr, ok := entry["command"].([]interface{})
	if !ok || len(cmdArr) != 3 || cmdArr[0] != "npx" || cmdArr[2] != ZAIMCPPackage {
		t.Errorf("command = %v, want [npx -y %s]", entry["command"], ZAIMCPPackage)
	}
	env := entry["environment"].(map[string]interface{})
	if env["Z_AI_API_KEY"] != "vision-key" {
		t.Errorf("unexpected environment: %+v", env)
	}

	configured, err := DetectMCPConfigured(home, "opencode")
	if err != nil || !configured {
		t.Fatalf("DetectMCPConfigured = %v, %v; want true, nil", configured, err)
	}

	if err := UnloadMCP(home, "opencode"); err != nil {
		t.Fatalf("UnloadMCP: %v", err)
	}
	configured, _ = DetectMCPConfigured(home, "opencode")
	if configured {
		t.Fatal("should not be configured after unload")
	}
}

// OpenCode's MCP config shares opencode.json with the provider config —
// confirm loading MCP doesn't disturb an existing provider entry.
func TestOpenCodeMCPPreservesProvider(t *testing.T) {
	home := t.TempDir()
	if err := Load(home, "opencode", PlanGlobal, "provider-key"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := LoadMCP(home, "opencode", "vision-key"); err != nil {
		t.Fatalf("LoadMCP: %v", err)
	}
	m := readJSON(t, Tools[1].ConfigPath(home))
	if _, ok := m["provider"]; !ok {
		t.Error("provider config lost after LoadMCP")
	}
	if _, ok := m["mcp"]; !ok {
		t.Error("mcp config missing")
	}
}

func TestCrushMCPLoadDetectUnload(t *testing.T) {
	home := t.TempDir()

	if err := LoadMCP(home, "crush", "vision-key"); err != nil {
		t.Fatalf("LoadMCP: %v", err)
	}

	m := readJSON(t, Tools[2].ConfigPath(home))
	servers := m["mcp"].(map[string]interface{})
	entry := servers[zaiMCPServerName].(map[string]interface{})
	if entry["type"] != "stdio" || entry["command"] != "npx" {
		t.Errorf("unexpected entry: %+v", entry)
	}

	configured, err := DetectMCPConfigured(home, "crush")
	if err != nil || !configured {
		t.Fatalf("DetectMCPConfigured = %v, %v; want true, nil", configured, err)
	}

	if err := UnloadMCP(home, "crush"); err != nil {
		t.Fatalf("UnloadMCP: %v", err)
	}
	configured, _ = DetectMCPConfigured(home, "crush")
	if configured {
		t.Fatal("should not be configured after unload")
	}
}

func TestCrushMCPPreservesProvider(t *testing.T) {
	home := t.TempDir()
	if err := Load(home, "crush", PlanGlobal, "provider-key"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := LoadMCP(home, "crush", "vision-key"); err != nil {
		t.Fatalf("LoadMCP: %v", err)
	}
	m := readJSON(t, Tools[2].ConfigPath(home))
	if _, ok := m["providers"]; !ok {
		t.Error("providers config lost after LoadMCP")
	}
}

func TestFactoryDroidMCPLoadDetectUnload(t *testing.T) {
	home := t.TempDir()

	if err := LoadMCP(home, "factory-droid", "vision-key"); err != nil {
		t.Fatalf("LoadMCP: %v", err)
	}

	mcpPath := mcpConfigPathFactoryDroid(home)
	m := readJSON(t, mcpPath)
	servers := m["mcpServers"].(map[string]interface{})
	entry := servers[zaiMCPServerName].(map[string]interface{})
	if entry["command"] != "npx" {
		t.Errorf("unexpected entry: %+v", entry)
	}
	if _, hasType := entry["type"]; hasType {
		t.Error("Factory Droid's real config has no \"type\" field — don't add one")
	}

	configured, err := DetectMCPConfigured(home, "factory-droid")
	if err != nil || !configured {
		t.Fatalf("DetectMCPConfigured = %v, %v; want true, nil", configured, err)
	}

	if err := UnloadMCP(home, "factory-droid"); err != nil {
		t.Fatalf("UnloadMCP: %v", err)
	}
	configured, _ = DetectMCPConfigured(home, "factory-droid")
	if configured {
		t.Fatal("should not be configured after unload")
	}
}

// Factory Droid's MCP config is a *separate* file (~/.factory/mcp.json) from
// its credential config (~/.factory/settings.json) — confirm LoadMCP never
// touches settings.json's customModels.
func TestFactoryDroidMCPDoesNotTouchSettingsJSON(t *testing.T) {
	home := t.TempDir()
	if err := Load(home, "factory-droid", PlanGlobal, "cred-key"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	settingsPath := Tools[3].ConfigPath(home)
	before := readJSON(t, settingsPath)

	if err := LoadMCP(home, "factory-droid", "vision-key"); err != nil {
		t.Fatalf("LoadMCP: %v", err)
	}
	after := readJSON(t, settingsPath)
	if len(before["customModels"].([]interface{})) != len(after["customModels"].([]interface{})) {
		t.Errorf("settings.json's customModels changed: before=%v after=%v", before["customModels"], after["customModels"])
	}
	if settingsPath == mcpConfigPathFactoryDroid(home) {
		t.Fatal("MCP config path must differ from the credential settings path")
	}
}

func TestCursorMCPLoadDetectUnload(t *testing.T) {
	home := t.TempDir()
	path := mcpConfigPathCursor(home)

	// Pre-existing unrelated config must survive the round-trip.
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	_ = os.WriteFile(path, []byte(`{"unrelated":true}`), 0o600)

	if err := LoadMCP(home, "cursor", "vision-key"); err != nil {
		t.Fatalf("LoadMCP: %v", err)
	}

	m := readJSON(t, path)
	if m["unrelated"] != true {
		t.Error("unrelated key lost")
	}
	servers := m["mcpServers"].(map[string]interface{})
	entry := servers[zaiMCPServerName].(map[string]interface{})
	if entry["command"] != "npx" {
		t.Errorf("unexpected entry: %+v", entry)
	}

	configured, err := DetectMCPConfigured(home, "cursor")
	if err != nil || !configured {
		t.Fatalf("DetectMCPConfigured = %v, %v; want true, nil", configured, err)
	}

	if err := UnloadMCP(home, "cursor"); err != nil {
		t.Fatalf("UnloadMCP: %v", err)
	}
	m = readJSON(t, path)
	if m["unrelated"] != true {
		t.Error("unrelated key lost after unload")
	}
	configured, _ = DetectMCPConfigured(home, "cursor")
	if configured {
		t.Fatal("should not be configured after unload")
	}
}

func TestDetectMCPConfiguredUnknownTool(t *testing.T) {
	if _, err := DetectMCPConfigured(t.TempDir(), "does-not-exist"); err == nil {
		t.Fatal("expected an error for an unsupported tool")
	}
}

func TestLoadMCPUnknownTool(t *testing.T) {
	if err := LoadMCP(t.TempDir(), "does-not-exist", "k"); err == nil {
		t.Fatal("expected an error for an unsupported tool")
	}
}
