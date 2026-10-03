package coding

import "path/filepath"

// ClaudeCodeID is Claude Code's tool ID — the one tool LoadConfig.Claude tunes.
const ClaudeCodeID = "claude-code"

// Claude Code: credentials go into the env block of ~/.claude/settings.json;
// onboarding state and MCP servers live in ~/.claude.json.
var claudeCode = Tool{
	ID:          ClaudeCodeID,
	Command:     "claude",
	DisplayName: "Claude Code",
	InstallHint: "npm install -g @anthropic-ai/claude-code",
	aliases:     []string{"claude"},
	configPath:  claudeSettingsPath,
	load:        loadClaudeCode,
	unload:      unloadClaudeCode,
	detect:      detectClaudeCode,
	mcp: mcpTarget{
		path:   claudeStatePath,
		key:    "mcpServers",
		local:  stdioEntry,
		remote: httpEntry,
	},
}

func claudeSettingsPath(home string) string { return filepath.Join(home, ".claude", "settings.json") }
func claudeStatePath(home string) string    { return filepath.Join(home, ".claude.json") }

// claudeManagedEnv lists every env var a plan config writes, so a reload
// starts clean and an unload removes exactly what was written.
var claudeManagedEnv = []string{
	"ANTHROPIC_AUTH_TOKEN",
	"ANTHROPIC_BASE_URL",
	"API_TIMEOUT_MS",
	"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC",
	"ANTHROPIC_DEFAULT_HAIKU_MODEL",
	"ANTHROPIC_DEFAULT_SONNET_MODEL",
	"ANTHROPIC_DEFAULT_OPUS_MODEL",
	"CLAUDE_CODE_AUTO_COMPACT_WINDOW",
	"MAX_THINKING_TOKENS",
	"CLAUDE_CODE_MAX_OUTPUT_TOKENS",
}

// loadClaudeCode mirrors the official helper: it marks onboarding complete,
// then writes ANTHROPIC_AUTH_TOKEN (sent as "Authorization: Bearer", which
// Z.AI's Anthropic endpoint expects — not ANTHROPIC_API_KEY), the plan's
// Anthropic base URL, the documented timeout and traffic settings, and the
// tuning in cfg.Claude.
func loadClaudeCode(home string, cfg LoadConfig) error {
	err := editConfigMap(claudeStatePath(home), func(m map[string]any) {
		if _, ok := m["hasCompletedOnboarding"].(bool); !ok {
			m["hasCompletedOnboarding"] = true
		}
	})
	if err != nil {
		return err
	}
	return editConfigMap(claudeSettingsPath(home), func(s map[string]any) {
		env := objectField(s, "env")
		deleteKeys(env, append(claudeManagedEnv, "ANTHROPIC_API_KEY")...)
		env["ANTHROPIC_AUTH_TOKEN"] = cfg.APIKey
		env["ANTHROPIC_BASE_URL"] = Region(cfg.Plan).AnthropicBaseURL()
		env["API_TIMEOUT_MS"] = "3000000"
		env["CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"] = 1
		for k, v := range cfg.Claude.env() {
			env[k] = v
		}
	})
}

func unloadClaudeCode(home string) error {
	d, err := detectClaudeCode(home)
	if err != nil {
		return err
	}
	if !d.Configured {
		return ErrNotConfigured
	}
	return editConfigMap(claudeSettingsPath(home), func(s map[string]any) {
		deleteFromObject(s, "env", claudeManagedEnv...)
	})
}

// detectClaudeCode reports a plan only when ANTHROPIC_BASE_URL points at a
// plan endpoint — a token set for another provider is not Z.AI's.
func detectClaudeCode(home string) (Detection, error) {
	s, err := readConfigMap(claudeSettingsPath(home))
	if err != nil {
		return Detection{}, err
	}
	env, _ := s["env"].(map[string]any)
	baseURL, _ := env["ANTHROPIC_BASE_URL"].(string)
	plan, ok := planFromBaseURL(baseURL)
	if !ok {
		return Detection{}, nil
	}
	key, _ := env["ANTHROPIC_AUTH_TOKEN"].(string)
	d := Detection{Configured: true, Plan: plan, APIKey: key}
	haiku, _ := env["ANTHROPIC_DEFAULT_HAIKU_MODEL"].(string)
	sonnet, _ := env["ANTHROPIC_DEFAULT_SONNET_MODEL"].(string)
	opus, _ := env["ANTHROPIC_DEFAULT_OPUS_MODEL"].(string)
	if haiku != "" || sonnet != "" || opus != "" {
		d.ModelMap = &ModelMap{Haiku: haiku, Sonnet: sonnet, Opus: opus}
	}
	return d, nil
}

func deleteKeys(m map[string]any, keys ...string) {
	for _, k := range keys {
		delete(m, k)
	}
}
