package coding

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Tool describes a supported coding app, mirroring @z_ai/coding-helper's
// SUPPORTED_TOOLS table: the CLI binary name, install command, display name,
// and the config file the helper writes.
type Tool struct {
	ID            string
	Command       string // CLI binary looked up on PATH to detect installation
	DisplayName   string
	InstallCommand string
	configPath    func(home string) string
}

// ConfigPath returns the absolute config file path for this tool under home.
func (t Tool) ConfigPath(home string) string {
	return t.configPath(home)
}

// IsInstalled reports whether the tool's CLI binary is on PATH.
func (t Tool) IsInstalled() bool {
	_, err := exec.LookPath(t.Command)
	return err == nil
}

// Tools is the ordered registry of supported coding tools.
var Tools = []Tool{
	{
		ID:             "claude-code",
		Command:        "claude",
		DisplayName:    "Claude Code",
		InstallCommand: "npm install -g @anthropic-ai/claude-code",
		configPath:     func(h string) string { return filepath.Join(h, ".claude", "settings.json") },
	},
	{
		ID:             "opencode",
		Command:        "opencode",
		DisplayName:    "OpenCode",
		InstallCommand: "npm install -g opencode-ai",
		configPath:     func(h string) string { return filepath.Join(h, ".config", "opencode", "opencode.json") },
	},
	{
		ID:             "crush",
		Command:        "crush",
		DisplayName:    "Crush",
		InstallCommand: "npm install -g @charmland/crush",
		configPath:     func(h string) string { return filepath.Join(h, ".config", "crush", "crush.json") },
	},
	{
		ID:             "factory-droid",
		Command:        "droid",
		DisplayName:    "Factory Droid",
		InstallCommand: factoryDroidInstall(),
		// The helper's FactoryDroidManager writes ~/.factory/settings.json
		// (SUPPORTED_TOOLS lists config.json, but the manager uses settings.json).
		configPath: func(h string) string { return filepath.Join(h, ".factory", "settings.json") },
	},
}

func factoryDroidInstall() string {
	if runtime.GOOS == "windows" {
		return "irm https://app.factory.ai/cli/windows | iex"
	}
	return "curl -fsSL https://app.factory.ai/cli | sh"
}

// FindTool resolves a tool by ID or alias (e.g. "claude" → "claude-code").
func FindTool(id string) (Tool, error) {
	switch id {
	case "claude", "claude-code":
		return Tools[0], nil
	case "opencode":
		return Tools[1], nil
	case "crush":
		return Tools[2], nil
	case "factory-droid", "droid", "factory":
		return Tools[3], nil
	}
	return Tool{}, fmt.Errorf("unsupported tool %q (supported: claude-code, opencode, crush, factory-droid)", id)
}
