package coding

import (
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

// ErrNotConfigured is returned by Tool.Unload when the tool's config holds no
// Z.AI plan configuration, so there is nothing to remove.
var ErrNotConfigured = errors.New("not configured for a Z.AI plan")

// LoadConfig is what a tool's config is written from.
type LoadConfig struct {
	Plan   string
	APIKey string
	// Claude tunes the Claude Code config; other tools ignore it.
	Claude ClaudeOptions
}

// NewLoadConfig returns a LoadConfig with the recommended defaults.
func NewLoadConfig(plan, apiKey string) LoadConfig {
	return LoadConfig{Plan: plan, APIKey: apiKey, Claude: DefaultClaudeOptions()}
}

// Detection is the result of probing a tool's config for a Z.AI plan.
type Detection struct {
	Configured bool      // a Z.AI plan configuration is present
	Plan       string    // PlanGlobal / PlanChina, or "" if it can't be told
	APIKey     string    // the key found in the config (may be empty)
	ModelMap   *ModelMap // Claude Code tier→model mapping, when present
}

// Tool is a supported coding app: how to find it, where its config lives,
// and how to write, remove, and detect the plan and MCP servers in it. Each
// tool's behavior lives in its own tool_*.go file.
type Tool struct {
	ID          string
	Command     string // binary looked up on PATH to detect installation
	DisplayName string
	InstallHint string
	aliases     []string
	configPath  func(home string) string
	load        func(home string, cfg LoadConfig) error
	unload      func(home string) error
	detect      func(home string) (Detection, error)
	mcp         mcpTarget
}

// Tools is the ordered registry of supported coding tools.
var Tools = []Tool{claudeCode, codex, openCode, crush, factoryDroid}

// FindTool resolves a tool by ID or alias (e.g. "claude" → "claude-code").
func FindTool(id string) (Tool, error) {
	for _, t := range Tools {
		if t.ID == id || slices.Contains(t.aliases, id) {
			return t, nil
		}
	}
	ids := make([]string, len(Tools))
	for i, t := range Tools {
		ids[i] = t.ID
	}
	return Tool{}, fmt.Errorf("unsupported tool %q (supported: %s)", id, strings.Join(ids, ", "))
}

// ConfigPath returns the tool's config file under home.
func (t Tool) ConfigPath(home string) string { return t.configPath(home) }

// IsInstalled reports whether the tool's binary is on PATH.
func (t Tool) IsInstalled() bool {
	_, err := exec.LookPath(t.Command)
	return err == nil
}

// Load writes the plan into the tool's config, preserving unrelated settings.
func (t Tool) Load(home string, cfg LoadConfig) error {
	if !IsValidPlan(cfg.Plan) {
		return fmt.Errorf("invalid plan %q", cfg.Plan)
	}
	if cfg.APIKey == "" {
		return errors.New("API key is required")
	}
	return t.load(home, cfg)
}

// Unload removes the plan from the tool's config; ErrNotConfigured when the
// config holds none (the user's own settings are never touched).
func (t Tool) Unload(home string) error { return t.unload(home) }

// Detect reports the tool's Z.AI plan configuration.
func (t Tool) Detect(home string) (Detection, error) { return t.detect(home) }
