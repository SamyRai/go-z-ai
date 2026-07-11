package coding

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// readJSON reads path into a map for assertions.
func readJSON(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return m
}

func TestClaudeCodeLoadDetectUnload(t *testing.T) {
	home := t.TempDir()

	if err := Load(home, "claude-code", PlanGlobal, "key-123"); err != nil {
		t.Fatalf("Load: %v", err)
	}

	// settings.json must use ANTHROPIC_AUTH_TOKEN, not ANTHROPIC_API_KEY.
	env := readJSON(t, filepath.Join(home, ".claude", "settings.json"))["env"].(map[string]interface{})
	if env["ANTHROPIC_AUTH_TOKEN"] != "key-123" {
		t.Errorf("AUTH_TOKEN not set: %v", env["ANTHROPIC_AUTH_TOKEN"])
	}
	if _, ok := env["ANTHROPIC_API_KEY"]; ok {
		t.Error("ANTHROPIC_API_KEY should be removed in favor of AUTH_TOKEN")
	}
	if env["ANTHROPIC_BASE_URL"] != "https://api.z.ai/api/anthropic" {
		t.Errorf("base url wrong: %v", env["ANTHROPIC_BASE_URL"])
	}

	// ~/.claude.json onboarding flag set.
	if v := readJSON(t, filepath.Join(home, ".claude.json"))["hasCompletedOnboarding"]; v != true {
		t.Errorf("onboarding flag = %v, want true", v)
	}

	d, err := Detect(home, "claude-code")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !d.Configured || d.Plan != PlanGlobal || d.APIKey != "key-123" {
		t.Fatalf("unexpected detection: %+v", d)
	}

	if err := Unload(home, "claude-code"); err != nil {
		t.Fatalf("Unload: %v", err)
	}
	d, _ = Detect(home, "claude-code")
	if d.Configured {
		t.Fatal("should not be configured after unload")
	}
}

func TestClaudeCodePreservesExistingEnv(t *testing.T) {
	home := t.TempDir()
	settings := filepath.Join(home, ".claude", "settings.json")
	_ = os.MkdirAll(filepath.Dir(settings), 0o700)
	_ = os.WriteFile(settings, []byte(`{"env":{"MY_VAR":"keep"},"foo":1}`), 0o600)

	if err := Load(home, "claude", PlanChina, "k"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	env := readJSON(t, settings)["env"].(map[string]interface{})
	if env["MY_VAR"] != "keep" {
		t.Error("existing env var not preserved")
	}
	m := readJSON(t, settings)
	if m["foo"] != float64(1) {
		t.Error("sibling keys not preserved")
	}
}

func TestOpenCodeLoadDetectUnload(t *testing.T) {
	home := t.TempDir()
	if err := Load(home, "opencode", PlanGlobal, "oc-key"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	m := readJSON(t, filepath.Join(home, ".config", "opencode", "opencode.json"))
	prov := m["provider"].(map[string]interface{})
	if _, ok := prov["zai-coding-plan"]; !ok {
		t.Error("zai-coding-plan provider missing")
	}
	if m["model"] != "zai-coding-plan/glm-4.6" {
		t.Errorf("model = %v", m["model"])
	}

	d, _ := Detect(home, "opencode")
	if !d.Configured || d.Plan != PlanGlobal || d.APIKey != "oc-key" {
		t.Fatalf("unexpected detection: %+v", d)
	}

	if err := Unload(home, "opencode"); err != nil {
		t.Fatalf("Unload: %v", err)
	}
	d, _ = Detect(home, "opencode")
	if d.Configured {
		t.Fatal("should not be configured after unload")
	}
}

func TestOpenCodeChinaUsesZhipuProvider(t *testing.T) {
	home := t.TempDir()
	if err := Load(home, "opencode", PlanChina, "k"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	m := readJSON(t, filepath.Join(home, ".config", "opencode", "opencode.json"))
	prov := m["provider"].(map[string]interface{})
	if _, ok := prov["zhipuai-coding-plan"]; !ok {
		t.Error("china plan should use zhipuai-coding-plan provider")
	}
}

func TestCrushLoadDetectUnload(t *testing.T) {
	home := t.TempDir()
	if err := Load(home, "crush", PlanGlobal, "c-key"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	zai := readJSON(t, filepath.Join(home, ".config", "crush", "crush.json"))["providers"].(map[string]interface{})["zai"].(map[string]interface{})
	if zai["base_url"] != "https://api.z.ai/api/coding/paas/v4" {
		t.Errorf("base_url = %v", zai["base_url"])
	}

	d, _ := Detect(home, "crush")
	if !d.Configured || d.APIKey != "c-key" {
		t.Fatalf("unexpected detection: %+v", d)
	}

	if err := Unload(home, "crush"); err != nil {
		t.Fatalf("Unload: %v", err)
	}
	d, _ = Detect(home, "crush")
	if d.Configured {
		t.Fatal("should not be configured after unload")
	}
}

func TestFactoryDroidLoadDetectUnload(t *testing.T) {
	home := t.TempDir()
	if err := Load(home, "factory-droid", PlanGlobal, "f-key"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	models := readJSON(t, filepath.Join(home, ".factory", "settings.json"))["customModels"].([]interface{})
	if len(models) != 2 {
		t.Fatalf("expected 2 custom models, got %d", len(models))
	}
	var sawAnthropic, sawOpenAI bool
	for _, e := range models {
		em := e.(map[string]interface{})
		dn := em["displayName"].(string)
		if dn == "GLM-4.7 [GLM Coding Plan Global] - Anthropic" {
			sawAnthropic = true
		}
		if dn == "GLM-4.7 [GLM Coding Plan Global] - Openai" {
			sawOpenAI = true
		}
	}
	if !sawAnthropic || !sawOpenAI {
		t.Fatalf("missing model variants; anthropic=%v openai=%v", sawAnthropic, sawOpenAI)
	}

	d, _ := Detect(home, "factory-droid")
	if !d.Configured || d.APIKey != "f-key" {
		t.Fatalf("unexpected detection: %+v", d)
	}

	if err := Unload(home, "factory-droid"); err != nil {
		t.Fatalf("Unload: %v", err)
	}
	d, _ = Detect(home, "factory-droid")
	if d.Configured {
		t.Fatal("should not be configured after unload")
	}
}

func TestFactoryDroidPreservesOtherModels(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".factory", "settings.json")
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	_ = os.WriteFile(path, []byte(`{"customModels":[{"displayName":"My Custom","model":"x"}]}`), 0o600)

	if err := Load(home, "factory-droid", PlanGlobal, "k"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	models := readJSON(t, path)["customModels"].([]interface{})
	if len(models) != 3 { // 1 existing + 2 GLM
		t.Fatalf("expected 3 models (1 preserved + 2 GLM), got %d", len(models))
	}
}

func TestCursorLoadDetectUnload(t *testing.T) {
	home := t.TempDir()
	// The settings path is OS-dependent (Application Support on darwin,
	// ~/.cursor or ~/.config/Cursor elsewhere) — resolve it like Load does.
	tool, err := FindTool("cursor")
	if err != nil {
		t.Fatalf("FindTool: %v", err)
	}
	path := tool.ConfigPath(home)

	// Pre-existing unrelated settings must survive the round-trip.
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	_ = os.WriteFile(path, []byte(`{"editor.fontSize":14}`), 0o600)

	if err := Load(home, "cursor", PlanGlobal, "c-key"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	m := readJSON(t, path)
	if m["apiKey"] != "c-key" {
		t.Errorf("apiKey = %v, want c-key", m["apiKey"])
	}
	if m["baseURL"] != "https://api.z.ai/api/anthropic" {
		t.Errorf("baseURL = %v", m["baseURL"])
	}

	d, err := Detect(home, "cursor")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !d.Configured || d.Plan != PlanGlobal || d.APIKey != "c-key" {
		t.Fatalf("unexpected detection: %+v", d)
	}

	if err := Unload(home, "cursor"); err != nil {
		t.Fatalf("Unload: %v", err)
	}
	d, _ = Detect(home, "cursor")
	if d.Configured {
		t.Fatal("should not be configured after unload")
	}
	if m := readJSON(t, path); m["editor.fontSize"] != float64(14) {
		t.Errorf("unrelated setting not preserved: %v", m["editor.fontSize"])
	}
}

func TestFindToolAliases(t *testing.T) {
	for _, alias := range []string{"claude", "claude-code", "opencode", "crush", "factory-droid", "droid", "cursor"} {
		if _, err := FindTool(alias); err != nil {
			t.Errorf("FindTool(%q): %v", alias, err)
		}
	}
	if _, err := FindTool("nope"); err == nil {
		t.Error("expected error for unknown tool")
	}
}

// DefaultClaudeOptions writes Z.AI's recommended model mapping + a 1M
// auto-compact window; plain LoadClaudeCode writes neither (chelper-exact).
func TestClaudeCodeDefaultOptionsTuning(t *testing.T) {
	home := t.TempDir()
	if err := LoadClaudeCodeOpts(home, PlanGlobal, "k", DefaultClaudeOptions()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	env := readJSON(t, filepath.Join(home, ".claude", "settings.json"))["env"].(map[string]interface{})
	if env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] != "glm-4.5-air" {
		t.Errorf("haiku model = %v", env["ANTHROPIC_DEFAULT_HAIKU_MODEL"])
	}
	if env["ANTHROPIC_DEFAULT_SONNET_MODEL"] != "glm-4.7" || env["ANTHROPIC_DEFAULT_OPUS_MODEL"] != "glm-4.7" {
		t.Errorf("sonnet/opus mapping wrong: %v / %v", env["ANTHROPIC_DEFAULT_SONNET_MODEL"], env["ANTHROPIC_DEFAULT_OPUS_MODEL"])
	}
	if env["CLAUDE_CODE_AUTO_COMPACT_WINDOW"] != "1000000" {
		t.Errorf("auto-compact-window = %v", env["CLAUDE_CODE_AUTO_COMPACT_WINDOW"])
	}
	if _, ok := env["MAX_THINKING_TOKENS"]; ok {
		t.Error("MAX_THINKING_TOKENS should be omitted by default")
	}
}

func TestClaudeCodePlainLoadHasNoTuning(t *testing.T) {
	home := t.TempDir()
	if err := LoadClaudeCode(home, PlanGlobal, "k"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	env := readJSON(t, filepath.Join(home, ".claude", "settings.json"))["env"].(map[string]interface{})
	for _, k := range []string{
		"ANTHROPIC_DEFAULT_HAIKU_MODEL",
		"ANTHROPIC_DEFAULT_SONNET_MODEL",
		"ANTHROPIC_DEFAULT_OPUS_MODEL",
		"CLAUDE_CODE_AUTO_COMPACT_WINDOW",
	} {
		if _, ok := env[k]; ok {
			t.Errorf("plain load should not write %s", k)
		}
	}
}

func TestClaudeCodeCustomTuning(t *testing.T) {
	home := t.TempDir()
	opts := ClaudeOptions{
		ModelMap:          &ModelMap{Haiku: "glm-4.5-flash", Sonnet: "glm-4.6", Opus: "glm-5.2"},
		AutoCompactWindow: 128000,
		MaxThinkingTokens: 8000,
		MaxOutputTokens:   65536,
	}
	if err := LoadClaudeCodeOpts(home, PlanGlobal, "k", opts); err != nil {
		t.Fatalf("Load: %v", err)
	}
	env := readJSON(t, filepath.Join(home, ".claude", "settings.json"))["env"].(map[string]interface{})
	if env["ANTHROPIC_DEFAULT_OPUS_MODEL"] != "glm-5.2" {
		t.Errorf("opus = %v", env["ANTHROPIC_DEFAULT_OPUS_MODEL"])
	}
	if env["CLAUDE_CODE_AUTO_COMPACT_WINDOW"] != "128000" {
		t.Errorf("auto-compact = %v", env["CLAUDE_CODE_AUTO_COMPACT_WINDOW"])
	}
	if env["MAX_THINKING_TOKENS"] != "8000" {
		t.Errorf("max-thinking = %v", env["MAX_THINKING_TOKENS"])
	}
	if env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] != "65536" {
		t.Errorf("max-output = %v", env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"])
	}

	// Unload must remove all tuning vars, not just the base set.
	if err := UnloadClaudeCode(home); err != nil {
		t.Fatalf("Unload: %v", err)
	}
	m := readJSON(t, filepath.Join(home, ".claude", "settings.json"))
	if v, ok := m["env"]; ok && v != nil {
		t.Errorf("env should be fully removed after unload, got %v", v)
	}
}
